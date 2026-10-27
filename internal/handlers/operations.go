package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// =============================================================================
//  OperationsHandler — 医院运营分析（后台运营 / 全院经营核算）
// =============================================================================
//  该模块为管理层提供全院的量化视图：门诊量、医疗费/药费收入、科室与医生效能、
//  设备与床位资源的实时占用情况，以及药品库存预警。
//
//  设计要点：
//   1. 跨数据库兼容（SQLite / Postgres / MySQL）—— 刻意避免使用各库方言的日期
//      函数（如 strftime / date_trunc），改为在 Go 侧计算时间边界并按天聚合，
//      从而在三种驱动下行为完全一致。
//   2. 只读聚合：不修改任何业务数据，按 report:read 权限受控。
//   3. 查询窗口有默认上限，避免大表全量扫描。
// =============================================================================

// OperationsHandler serves read-only hospital-wide aggregates for administrators.
type OperationsHandler struct {
	db *gorm.DB
}

// RegisterOperations mounts the operations/analytics routes.
func RegisterOperations(r gin.IRouter, db *gorm.DB) {
	h := &OperationsHandler{db: db}
	// All analytics endpoints require report:read (granted to administrators).
	rep := r.Group("")
	rep.Use(middleware.RequireAny(db, "report:read"))
	{
		rep.GET("/operations/overview", h.Overview)
		rep.GET("/operations/trend", h.Trend)
		rep.GET("/operations/departments", h.Departments)
		rep.GET("/operations/doctors", h.Doctors)
		rep.GET("/operations/status", h.Status)
		rep.GET("/operations/revenue", h.Revenue)
	}
}

// ---------------------------------------------------------------------------
//  通用工具
// ---------------------------------------------------------------------------

// parseDays reads the ?days= query parameter, clamping it to [1, 365].
func parseDays(c *gin.Context, def int) int {
	days := c.DefaultQuery("days", "")
	if days == "" {
		return def
	}
	n := 0
	for _, ch := range days {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 || n > 365 {
		return def
	}
	return n
}

// startOfToday returns local midnight for today.
func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// dayKey formats a timestamp as YYYY-MM-DD (the grouping key for time series).
func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// sumFloat safely accumulates a float64 slice.
func sumFloat(ns []float64) float64 {
	var s float64
	for _, n := range ns {
		s += n
	}
	return s
}

// ===========================================================================
//  1. 全局概览（院长看板核心 KPI）
// ===========================================================================

// Overview returns headline hospital KPIs: volumes, today's activity, revenue,
// resource utilisation and stock warnings — everything an administrator needs on
// one screen.
func (h *OperationsHandler) Overview(c *gin.Context) {
	today := startOfToday()
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())

	var (
		totalPatients     int64
		totalDoctors      int64
		todayVisits       int64
		todayPresc        int64
		pendingSurgeries  int64
		pendingImaging    int64
		activeRooms       int64
		onlinePrinters    int64
		lowStockMedicines int64
		todayAppts        int64
		pendingAppts      int64
		totalBeds         int64
		occupiedBeds      int64
		pendingInvoices   int64
	)

	// 规模指标：累计患者数、在岗医生数。
	h.db.Model(&models.Patient{}).Count(&totalPatients)
	h.db.Model(&models.User{}).Where("role = ? AND status = ?", models.RoleDoctor, "active").Count(&totalDoctors)

	// 今日门诊量：以病例（接诊记录）为准，按 visit_date 落在今天统计。
	h.db.Model(&models.MedicalCase{}).Where("visit_date >= ?", today).Count(&todayVisits)
	// 今日处方数。
	h.db.Model(&models.Prescription{}).Where("created_at >= ?", today).Count(&todayPresc)

	// 预约挂号：今日排班量与仍待到诊的预约数。
	h.db.Model(&models.Appointment{}).Where("slot_at >= ? AND slot_at < ?", today, today.AddDate(0, 0, 1)).Count(&todayAppts)
	h.db.Model(&models.Appointment{}).Where("status = ?", "booked").Count(&pendingAppts)

	// 待处理：尚未开台的手术、尚未出具报告的影像检查。
	h.db.Model(&models.Surgery{}).Where("status = ?", "planned").Count(&pendingSurgeries)
	h.db.Model(&models.ImagingStudy{}).Where("status <> ?", "reported").Count(&pendingImaging)

	// 资源：可用房间、在线打印机。
	h.db.Model(&models.Room{}).Where("status = ?", "active").Count(&activeRooms)
	h.db.Model(&models.Printer{}).Where("status = ?", "active").Count(&onlinePrinters)

	// 药品库存：低于预警阈值的品种数。
	// 注意：必须要求 reorder_level > 0，否则从未设置预警线的药品
	// （默认 stock_qty=0、reorder_level=0）会被 0<=0 误判为缺货。
	h.db.Model(&models.Medicine{}).
		Where("is_active = ? AND reorder_level > 0 AND stock_qty <= reorder_level", true).
		Count(&lowStockMedicines)

	// 住院资源：总床位与已占用床位（占用率由前端计算）。
	h.db.Model(&models.Bed{}).Count(&totalBeds)
	h.db.Model(&models.Bed{}).Where("status = ?", "occupied").Count(&occupiedBeds)

	// 收费：待结算（pending）单据数量。
	h.db.Model(&models.Invoice{}).Where("status = ?", "pending").Count(&pendingInvoices)

	// 收入核算：处方金额（药费口径）与结算单实收（综合口径）。
	var todayRevenueRaw, monthRevenueRaw []float64
	h.db.Model(&models.Prescription{}).Where("created_at >= ?", today).Pluck("total_amount", &todayRevenueRaw)
	h.db.Model(&models.Prescription{}).Where("created_at >= ?", monthStart).Pluck("total_amount", &monthRevenueRaw)

	// 实收金额以 paid_at 落账时间为准，退款单不计入收入。
	var todayPaidRaw, monthPaidRaw []float64
	h.db.Model(&models.Invoice{}).
		Where("status = ? AND paid_at >= ?", "paid", today).
		Pluck("total", &todayPaidRaw)
	h.db.Model(&models.Invoice{}).
		Where("status = ? AND paid_at >= ?", "paid", monthStart).
		Pluck("total", &monthPaidRaw)

	// 本月退费金额（用于计算净收入）。
	var monthRefundRaw []float64
	h.db.Model(&models.Invoice{}).
		Where("status = ? AND updated_at >= ?", "refunded", monthStart).
		Pluck("total", &monthRefundRaw)

	occupancy := 0.0
	if totalBeds > 0 {
		occupancy = float64(occupiedBeds) / float64(totalBeds) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"patients":             totalPatients,
		"doctors":              totalDoctors,
		"visits_today":         todayVisits,
		"prescriptions_today":  todayPresc,
		"revenue_today":        round2(sumFloat(todayRevenueRaw)),
		"revenue_month":        round2(sumFloat(monthRevenueRaw)),
		"paid_today":           round2(sumFloat(todayPaidRaw)),
		"paid_month":           round2(sumFloat(monthPaidRaw)),
		"net_revenue_month":    round2(sumFloat(monthPaidRaw) - sumFloat(monthRefundRaw)),
		"pending_surgeries":    pendingSurgeries,
		"pending_imaging":      pendingImaging,
		"rooms_active":         activeRooms,
		"printers_online":      onlinePrinters,
		"low_stock":            lowStockMedicines,
		"appointments_today":   todayAppts,
		"appointments_pending": pendingAppts,
		"beds_total":           totalBeds,
		"beds_occupied":        occupiedBeds,
		"bed_occupancy":        round2(occupancy),
		"invoices_pending":     pendingInvoices,
		"currency":             "CNY",
	})
}

// ===========================================================================
//  2. 趋势（门诊量 / 收入 时间序列）
// ===========================================================================

// Trend returns per-day visit counts and revenue for the last N days (default
// 30), filling days with no activity so charts render a continuous axis.
func (h *OperationsHandler) Trend(c *gin.Context) {
	days := parseDays(c, 30)
	today := startOfToday()
	start := today.AddDate(0, 0, -(days - 1))

	// --- 按天聚合：取出窗口内的原始数据后在 Go 侧分桶（跨数据库一致） ---
	var cases []struct {
		CreatedAt time.Time
	}
	h.db.Model(&models.MedicalCase{}).Where("visit_date >= ?", start).Select("visit_date as created_at").Scan(&cases)

	var rx []struct {
		CreatedAt time.Time
		Amount    float64
	}
	h.db.Model(&models.Prescription{}).
		Where("created_at >= ?", start).
		Select("created_at, total_amount as amount").
		Scan(&rx)

	// 预填连续日期轴，保证前端图表不出现断点。
	visitByDay := map[string]int64{}
	revByDay := map[string]float64{}
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		visitByDay[dayKey(d)] = 0
		revByDay[dayKey(d)] = 0
	}
	for _, cs := range cases {
		if k := dayKey(cs.CreatedAt); k >= dayKey(start) && k <= dayKey(today) {
			visitByDay[k]++
		}
	}
	for _, r := range rx {
		if k := dayKey(r.CreatedAt); k >= dayKey(start) && k <= dayKey(today) {
			revByDay[k] += r.Amount
		}
	}

	series := make([]gin.H, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		k := dayKey(d)
		series = append(series, gin.H{
			"date":    k,
			"visits":  visitByDay[k],
			"revenue": round2(revByDay[k]),
		})
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "series": series})
}

// ===========================================================================
//  3. 科室效能
// ===========================================================================

// Departments aggregates per-department performance: doctor headcount, visit
// volume and generated revenue. Departments come from room/doctor assignments.
func (h *OperationsHandler) Departments(c *gin.Context) {
	// 科室字典：以医生所属科室为准。
	var depts []struct {
		Department string
		Doctors    int64
	}
	h.db.Model(&models.User{}).
		Where("role = ? AND department <> ''", models.RoleDoctor).
		Select("department, count(*) as doctors").
		Group("department").
		Scan(&depts)

	var Today = startOfToday()
	out := make([]gin.H, 0, len(depts))
	for _, d := range depts {
		var visits int64
		var revenue []float64

		// 该科室医生产生的门诊量（通过 doctor_id 关联）。
		h.db.Model(&models.MedicalCase{}).
			Joins("join users on users.id = medical_cases.doctor_id").
			Where("users.department = ?", d.Department).
			Count(&visits)

		// 该科室的处方收入。
		h.db.Model(&models.Prescription{}).
			Joins("join users on users.id = prescriptions.doctor_id").
			Where("users.department = ?", d.Department).
			Pluck("prescriptions.total_amount", &revenue)

		out = append(out, gin.H{
			"department": d.Department,
			"doctors":    d.Doctors,
			"visits":     visits,
			"revenue":    round2(sumFloat(revenue)),
		})
	}
	// today is used to keep the intent explicit for future windowed stats.
	_ = Today
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// ===========================================================================
//  4. 医生效能排行
// ===========================================================================

// Doctors returns per-doctor productivity metrics, ordered by visit volume.
func (h *OperationsHandler) Doctors(c *gin.Context) {
	var docs []models.User
	h.db.Where("role = ?", models.RoleDoctor).Order("id").Find(&docs)

	items := make([]gin.H, 0, len(docs))
	for _, d := range docs {
		var visits int64
		var rxCount int64
		var revenue []float64

		h.db.Model(&models.MedicalCase{}).Where("doctor_id = ?", d.ID).Count(&visits)
		h.db.Model(&models.Prescription{}).Where("doctor_id = ?", d.ID).Count(&rxCount)
		h.db.Model(&models.Prescription{}).
			Where("doctor_id = ?", d.ID).
			Pluck("total_amount", &revenue)

		items = append(items, gin.H{
			"doctor_id":     d.ID,
			"name":          d.Name,
			"department":    d.Department,
			"title":         d.Title,
			"visits":        visits,
			"prescriptions": rxCount,
			"revenue":       round2(sumFloat(revenue)),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ===========================================================================
//  5. 资源实时状态
// ===========================================================================

// Status gives a live snapshot of physical resources: rooms (free / occupied by
// today's consultations), printers and their recent print-job audit trail.
func (h *OperationsHandler) Status(c *gin.Context) {
	today := startOfToday()

	var rooms []models.Room
	h.db.Where("status = ?", "active").Order("room_no").Find(&rooms)

	// 今日各房间已完成的接诊量，用于估算占用情况。
	type roomStat struct {
		RoomNo string
		Cnt    int64
	}
	var stats []roomStat
	h.db.Model(&models.MedicalCase{}).
		Where("visit_date >= ?", today).
		Select("room_no, count(*) as cnt").
		Group("room_no").
		Scan(&stats)
	busy := map[string]int64{}
	for _, s := range stats {
		busy[s.RoomNo] = s.Cnt
	}

	roomItems := make([]gin.H, 0, len(rooms))
	for _, r := range rooms {
		roomItems = append(roomItems, gin.H{
			"id":           r.ID,
			"room_no":      r.RoomNo,
			"name_zh":      r.NameZH,
			"type":         r.Type,
			"doctor_name":  r.DoctorName,
			"visits_today": busy[r.RoomNo],
			"occupied":     busy[r.RoomNo] > 0,
		})
	}

	var printers []models.Printer
	h.db.Order("is_default desc, id").Find(&printers)

	var recentJobs []models.PrintJob
	h.db.Order("id desc").Limit(10).Find(&recentJobs)

	// 住院床位按病区聚合，附带占用率，供管理员查看床位周转。
	type wardStat struct {
		WardID   uint
		WardName string
		Total    int64
		Occupied int64
	}
	var wardStats []wardStat
	h.db.Model(&models.Bed{}).
		Select("ward_id, ward_name, count(*) as total, sum(case when status = ? then 1 else 0 end) as occupied", "occupied").
		Group("ward_id, ward_name").
		Scan(&wardStats)

	bedItems := make([]gin.H, 0, len(wardStats))
	for _, w := range wardStats {
		rate := 0.0
		if w.Total > 0 {
			rate = float64(w.Occupied) / float64(w.Total) * 100
		}
		bedItems = append(bedItems, gin.H{
			"ward_id":   w.WardID,
			"ward_name": w.WardName,
			"total":     w.Total,
			"occupied":  w.Occupied,
			"occupancy": round2(rate),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms":       roomItems,
		"printers":    printers,
		"recent_jobs": recentJobs,
		"beds":        bedItems,
	})
}

// ===========================================================================
//  6. 收入构成分析
// ===========================================================================

// Revenue breaks down the hospital's billing income for the requested window.
//
// It returns three views that administrators actually reconcile against:
//   - by_status  : how much money sits in each invoice state (pending/paid/...)
//   - by_method  : cash vs card vs mobile payment vs insurance
//   - by_item    : which charge categories (consult/medicine/exam/surgery/bed) drive revenue
//
// Only paid invoices contribute to "revenue"; the rest are reported separately
// so receivables remain visible.
func (h *OperationsHandler) Revenue(c *gin.Context) {
	days := parseDays(c, 30)
	today := startOfToday()
	start := today.AddDate(0, 0, -(days - 1))

	// --- 按结算单状态统计金额与笔数 ---
	type statusRow struct {
		Status string
		Count  int64
		Amount float64
	}
	var statusRows []statusRow
	h.db.Model(&models.Invoice{}).
		Where("created_at >= ?", start).
		Select("status, count(*) as count, sum(total) as amount").
		Group("status").
		Scan(&statusRows)

	byStatus := make([]gin.H, 0, len(statusRows))
	var totalBilled, totalPaid float64
	for _, r := range statusRows {
		byStatus = append(byStatus, gin.H{
			"status": r.Status,
			"count":  r.Count,
			"amount": round2(r.Amount),
		})
		totalBilled += r.Amount
		if r.Status == "paid" {
			totalPaid += r.Amount
		}
	}

	// --- 按支付方式统计（仅已收款） ---
	type methodRow struct {
		Method string
		Count  int64
		Amount float64
	}
	var methodRows []methodRow
	h.db.Model(&models.Invoice{}).
		Where("status = ? AND paid_at >= ?", "paid", start).
		Select("payment_method as method, count(*) as count, sum(total) as amount").
		Group("payment_method").
		Scan(&methodRows)

	byMethod := make([]gin.H, 0, len(methodRows))
	for _, r := range methodRows {
		byMethod = append(byMethod, gin.H{
			"method": r.Method,
			"count":  r.Count,
			"amount": round2(r.Amount),
		})
	}

	// --- 按收费项目类别统计（基于已收款结算单的明细） ---
	type itemRow struct {
		ItemType string
		Count    int64
		Amount   float64
	}
	var itemRows []itemRow
	h.db.Model(&models.InvoiceItem{}).
		Joins("join invoices on invoices.id = invoice_items.invoice_id").
		Where("invoices.status = ? AND invoices.paid_at >= ?", "paid", start).
		Select("invoice_items.item_type as item_type, count(*) as count, sum(invoice_items.amount) as amount").
		Group("invoice_items.item_type").
		Scan(&itemRows)

	byItem := make([]gin.H, 0, len(itemRows))
	for _, r := range itemRows {
		byItem = append(byItem, gin.H{
			"item_type": r.ItemType,
			"count":     r.Count,
			"amount":    round2(r.Amount),
		})
	}

	// 退费金额单独列出，避免与实收混淆。
	var refundRaw []float64
	h.db.Model(&models.Invoice{}).
		Where("status = ? AND updated_at >= ?", "refunded", start).
		Pluck("total", &refundRaw)

	c.JSON(http.StatusOK, gin.H{
		"days":         days,
		"total_billed": round2(totalBilled),
		"total_paid":   round2(totalPaid),
		"total_refund": round2(sumFloat(refundRaw)),
		"by_status":    byStatus,
		"by_method":    byMethod,
		"by_item":      byItem,
		"currency":     "CNY",
	})
}

// round2 rounds to two decimals for money-safe presentation.
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
