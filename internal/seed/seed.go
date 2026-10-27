package seed

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
)

func hash(p string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("seed: hash password: %v", err)
	}
	return string(h)
}

// permissionDef describes one entry in the permission catalog.
type permissionDef struct {
	Code        string
	NameZH      string
	NameEN      string
	Group       string
	Description string
}

// permissionCatalog is the full set of capabilities the system understands.
var permissionCatalog = []permissionDef{
	{"dashboard:view", "查看工作台", "View dashboard", "仪表盘", "访问系统工作台"},
	{"patient:read", "查看患者", "View patients", "患者", "浏览患者档案"},
	{"patient:write", "管理患者", "Manage patients", "患者", "新建/编辑患者"},
	{"case:read", "查看病例", "View cases", "病例", "浏览诊疗记录"},
	{"case:write", "管理病例", "Manage cases", "病例", "新建/编辑病例"},
	{"prescription:read", "查看处方", "View prescriptions", "处方", "浏览处方"},
	{"prescription:create", "开具处方", "Create prescriptions", "处方", "开具处方"},
	{"prescription:dispense", "发药", "Dispense medicine", "处方", "处方发药"},
	{"medicine:read", "查看药品", "View medicines", "药品", "浏览药品库"},
	{"medicine:manage", "管理药品", "Manage medicines", "药品", "增删改药品与定价"},
	{"surgery:read", "查看手术", "View surgeries", "手术", "浏览手术记录"},
	{"surgery:manage", "管理手术", "Manage surgeries", "手术", "排期/记录手术"},
	{"imaging:read", "查看影像", "View imaging", "影像", "浏览影像检查"},
	{"imaging:upload", "管理影像", "Manage imaging", "影像", "开单/上传影像"},
	{"imaging:report", "出具报告", "Report imaging", "影像", "书写影像报告"},
	{"room:read", "查看房间", "View rooms", "房间", "浏览房间"},
	{"room:manage", "管理房间", "Manage rooms", "房间", "管理门诊房间"},
	{"printer:read", "查看打印机", "View printers", "打印机", "浏览打印机"},
	{"printer:manage", "管理打印机", "Manage printers", "打印机", "管理打印机"},
	{"inventory:read", "查看库存", "View inventory", "库存", "浏览药品库存"},
	{"inventory:manage", "管理库存", "Manage inventory", "库存", "药品入库/出库与盘点"},
	{"report:read", "运营报表", "Operations reports", "运营", "查看全院运营分析与报表"},
	{"appointment:read", "查看预约", "View appointments", "预约", "浏览预约挂号"},
	{"appointment:manage", "管理预约", "Manage appointments", "预约", "预约建档/改期/叫号"},
	{"bed:read", "查看床位", "View beds", "住院", "浏览病区与床位"},
	{"bed:manage", "管理床位", "Manage beds", "住院", "入院/出院/床位调配"},
	{"billing:read", "查看收费", "View billing", "收费", "浏览结算单"},
	{"billing:manage", "管理收费", "Manage billing", "收费", "开单/收费/退费"},
	{"user:read", "查看账号", "View users", "账号", "浏览用户账号"},
	{"user:manage", "管理账号", "Manage users", "账号", "创建/停用账号"},
	{"role:read", "查看角色", "View roles", "角色", "浏览角色"},
	{"role:manage", "管理角色", "Manage roles", "角色", "分配权限/角色"},
}

// roleDef maps a role to the permissions it grants by default.
type roleDef struct {
	Code        string
	NameZH      string
	NameEN      string
	Description string
	IsSystem    bool
	Sort        int
	Permissions []string
}

var roleDefs = []roleDef{
	{
		Code: models.RoleAdmin, NameZH: "超级管理员", NameEN: "Administrator",
		Description: "拥有系统全部权限", IsSystem: true, Sort: 0,
		Permissions: allCodes(),
	},
	{
		Code: models.RoleDoctor, NameZH: "医生", NameEN: "Doctor",
		Description: "诊疗、开方、查看影像", IsSystem: true, Sort: 1,
		Permissions: []string{
			"dashboard:view", "patient:read", "patient:write", "case:read", "case:write",
			"prescription:read", "prescription:create", "medicine:read", "surgery:read",
			"imaging:read", "imaging:upload", "room:read", "printer:read",
			"appointment:read", "billing:read",
		},
	},
	{
		Code: models.RoleNurse, NameZH: "护士", NameEN: "Nurse",
		Description: "协助护理、查看患者与处方", IsSystem: true, Sort: 2,
		Permissions: []string{
			"dashboard:view", "patient:read", "patient:write", "case:read",
			"prescription:read", "medicine:read", "imaging:read", "room:read",
			"appointment:read", "bed:read", "bed:manage",
		},
	},
	{
		Code: models.RoleReceptionist, NameZH: "挂号收费", NameEN: "Receptionist",
		Description: "挂号、收费、患者登记", IsSystem: true, Sort: 3,
		Permissions: []string{
			"dashboard:view", "patient:read", "patient:write", "prescription:read",
			"medicine:read", "room:read", "printer:read",
			"appointment:read", "appointment:manage", "billing:read", "billing:manage",
			"bed:read",
		},
	},
	{
		Code: models.RoleRadiologist, NameZH: "影像科", NameEN: "Radiologist",
		Description: "影像检查与报告", IsSystem: true, Sort: 4,
		Permissions: []string{
			"dashboard:view", "patient:read", "imaging:read", "imaging:upload",
			"imaging:report", "room:read",
		},
	},
	{
		Code: models.RoleSurgeon, NameZH: "手术医生", NameEN: "Surgeon",
		Description: "手术排期与记录", IsSystem: true, Sort: 5,
		Permissions: []string{
			"dashboard:view", "patient:read", "case:read", "surgery:read",
			"surgery:manage", "imaging:read", "room:read",
		},
	},
	{
		Code: models.RolePharmacist, NameZH: "药房", NameEN: "Pharmacist",
		Description: "处方发药与药品查看", IsSystem: true, Sort: 6,
		Permissions: []string{
			"dashboard:view", "patient:read", "prescription:read", "prescription:dispense",
			"medicine:read", "room:read", "printer:read",
			"inventory:read", "inventory:manage",
		},
	},
}

func allCodes() []string {
	codes := make([]string, 0, len(permissionCatalog))
	for _, p := range permissionCatalog {
		codes = append(codes, p.Code)
	}
	return codes
}

// SeedRBAC creates the permission catalog and the default role/permission map.
func SeedRBAC(db *gorm.DB) {
	// 1. Permission catalog (idempotent upsert by code).
	for _, p := range permissionCatalog {
		var existing models.Permission
		if err := db.Where("code = ?", p.Code).First(&existing).Error; err != nil {
			db.Create(&models.Permission{
				Code: p.Code, NameZH: p.NameZH, NameEN: p.NameEN, Group: p.Group, Description: p.Description,
			})
		}
	}
	// 2. Roles + their permission grants.
	for _, r := range roleDefs {
		var role models.Role
		if err := db.Where("code = ?", r.Code).First(&role).Error; err != nil {
			role = models.Role{
				Code: r.Code, NameZH: r.NameZH, NameEN: r.NameEN,
				Description: r.Description, IsSystem: r.IsSystem, Sort: r.Sort,
			}
			db.Create(&role)
		} else {
			role.NameZH = r.NameZH
			role.NameEN = r.NameEN
			role.Description = r.Description
			role.IsSystem = r.IsSystem
			role.Sort = r.Sort
			db.Save(&role)
		}
		// Replace permission grants.
		db.Where("role_code = ?", r.Code).Delete(&models.RolePermission{})
		bulk := make([]models.RolePermission, 0, len(r.Permissions))
		for _, code := range r.Permissions {
			bulk = append(bulk, models.RolePermission{RoleCode: r.Code, PermissionCode: code})
		}
		db.Create(&bulk)
	}
	log.Printf("seed: RBAC ready (%d permissions, %d roles)", len(permissionCatalog), len(roleDefs))
}

// SeedUsers creates the initial admin/doctor accounts and, when seedDemo is set,
// a representative user for every role so RBAC can be exercised immediately.
func SeedUsers(db *gorm.DB, seedDemo bool) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}
	// 默认超级管理员：root / root（首次登录后请务必修改密码）。
	// 其余为各岗位演示账号，随 SEED_DEMO=false 可不注入。
	users := []models.User{
		{Username: "root", PasswordHash: hash("root"), Name: "系统管理员", Role: models.RoleAdmin, Department: "信息科", Title: "超级管理员", Status: "active"},
		{Username: "doctor", PasswordHash: hash("doctor123"), Name: "张医生", Role: models.RoleDoctor, Department: "内科", Title: "主治医师", Status: "active"},
	}
	if seedDemo {
		users = append(users,
			models.User{Username: "nurse", PasswordHash: hash("nurse123"), Name: "李护士", Role: models.RoleNurse, Department: "内科", Title: "主管护师", Status: "active"},
			models.User{Username: "reception", PasswordHash: hash("recept123"), Name: "王挂号", Role: models.RoleReceptionist, Department: "门诊", Title: "挂号员", Status: "active"},
			models.User{Username: "radiologist", PasswordHash: hash("radio123"), Name: "赵影像", Role: models.RoleRadiologist, Department: "影像科", Title: "主任医师", Status: "active"},
			models.User{Username: "surgeon", PasswordHash: hash("surgeon123"), Name: "陈手术", Role: models.RoleSurgeon, Department: "外科", Title: "主任医师", Status: "active"},
			models.User{Username: "pharmacist", PasswordHash: hash("pharm123"), Name: "刘药房", Role: models.RolePharmacist, Department: "药剂科", Title: "药师", Status: "active"},
		)
	}
	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("seed: create users: %v", err)
	}
	log.Printf("seed: created %d users (root/root, doctor/doctor123)", len(users))
}

// doctorID returns the id of the seeded doctor account (best-effort).
func doctorID(db *gorm.DB) uint {
	var u models.User
	if err := db.Where("username = ?", "doctor").First(&u).Error; err != nil {
		return 0
	}
	return u.ID
}

// SeedRooms populates sample outpatient / OR / radiology rooms.
func SeedRooms(db *gorm.DB, seedDemo bool) {
	var count int64
	db.Model(&models.Room{}).Count(&count)
	if count > 0 {
		return
	}
	docID := doctorID(db)
	rooms := []models.Room{
		{RoomNo: "201", NameZH: "内科诊室1", NameEN: "Internal Med 1", Type: "consultation", Department: "内科", Floor: "2F", DoctorID: docID, DoctorName: "张医生", Status: "active"},
		{RoomNo: "202", NameZH: "内科诊室2", NameEN: "Internal Med 2", Type: "consultation", Department: "内科", Floor: "2F", Status: "active"},
		{RoomNo: "OR1", NameZH: "手术室1", NameEN: "Operating Room 1", Type: "surgery", Department: "外科", Floor: "3F", Status: "active"},
		{RoomNo: "RD1", NameZH: "影像检查室", NameEN: "Radiology 1", Type: "imaging", Department: "影像科", Floor: "1F", Status: "active"},
		{RoomNo: "ER1", NameZH: "急诊抢救室", NameEN: "Emergency 1", Type: "emergency", Department: "急诊", Floor: "1F", Status: "active"},
	}
	if err := db.Create(&rooms).Error; err != nil {
		log.Fatalf("seed: create rooms: %v", err)
	}
	log.Printf("seed: created %d rooms", len(rooms))
}

// SeedWards creates the inpatient wards together with their beds so the
// occupancy dashboard and bed management page have meaningful baseline data.
//
// Each ward holds a small set of beds in different states (available /
// occupied / reserved / maintenance) which exercises the occupancy maths.
func SeedWards(db *gorm.DB, seedDemo bool) {
	var count int64
	db.Model(&models.Ward{}).Count(&count)
	if count > 0 {
		return
	}
	wards := []models.Ward{
		{Code: "W-IN", NameZH: "内科病区", NameEN: "Internal Medicine Ward", Department: "内科", Floor: "5F", Status: "active"},
		{Code: "W-SU", NameZH: "外科病区", NameEN: "Surgery Ward", Department: "外科", Floor: "6F", Status: "active"},
		{Code: "W-ICU", NameZH: "重症监护病房", NameEN: "Intensive Care Unit", Department: "重症医学科", Floor: "7F", Status: "active"},
	}
	if err := db.Create(&wards).Error; err != nil {
		log.Fatalf("seed: create wards: %v", err)
	}
	// 每个病区 10 张床：0-5 可用，6-8 占用，9 维护，形成可计算的占用率。
	bedCounts := map[string]int{"W-IN": 10, "W-SU": 10, "W-ICU": 6}
	var beds []models.Bed
	for _, w := range wards {
		n := bedCounts[w.Code]
		for i := 1; i <= n; i++ {
			bedNo := fmt.Sprintf("%02d", i)
			status := "available"
			daily := 80.0
			switch w.Code {
			case "W-ICU":
				daily = 480.0
				if i <= 4 {
					status = "occupied"
				} else if i == 5 {
					status = "reserved"
				}
			default:
				if i >= 7 && i <= 9 {
					status = "occupied"
				} else if i == 10 {
					status = "maintenance"
				}
			}
			beds = append(beds, models.Bed{
				WardID: w.ID, WardName: w.NameZH, BedNo: bedNo,
				Code: w.Code + "-" + bedNo, Status: status, DailyRate: daily,
			})
		}
	}
	if err := db.CreateInBatches(beds, 50).Error; err != nil {
		log.Fatalf("seed: create beds: %v", err)
	}
	log.Printf("seed: created %d wards / %d beds", len(wards), len(beds))
}

// SeedPrinters creates a doctor-bound thermal printer and a global default.
func SeedPrinters(db *gorm.DB, seedDemo bool) {
	var count int64
	db.Model(&models.Printer{}).Count(&count)
	if count > 0 {
		return
	}
	docID := doctorID(db)
	printers := []models.Printer{
		{Name: "201诊室热敏打印机", Type: "thermal", Connection: "network", IPAddress: "192.168.1.201", Port: "9100", DoctorID: docID, DoctorName: "张医生", RoomID: 0, RoomNo: "201", IsDefault: false, Status: "active"},
		{Name: "前台A4激光打印机", Type: "a4", Connection: "network", IPAddress: "192.168.1.10", Port: "9100", DoctorID: 0, DoctorName: "", RoomID: 0, RoomNo: "", IsDefault: true, Status: "active"},
	}
	if err := db.Create(&printers).Error; err != nil {
		log.Fatalf("seed: create printers: %v", err)
	}
	log.Printf("seed: created %d printers", len(printers))
}

// SeedMedicines populates the common drug formulary on first run.
func SeedMedicines(db *gorm.DB) {
	var count int64
	db.Model(&models.Medicine{}).Count(&count)
	if count > 0 {
		return
	}
	items := commonMedicines()
	// 赋予初始库存与补货警戒线，让运营看板的缺货预警在演示数据下即有真实表现：
	// 每隔第 7 个品种给出低于警戒线的库存以触发预警，其余保持充足库存。
	// 直接在入库前赋值，避免批量 Update 的零值更新陷阱。
	for i := range items {
		items[i].ReorderLevel = 20
		items[i].StockQty = 200 - float64(i%5)*10
		if i%7 == 0 {
			items[i].StockQty = float64(5 + i%10)
		}
	}
	if err := db.CreateInBatches(items, 50).Error; err != nil {
		log.Fatalf("seed: create medicines: %v", err)
	}
	log.Printf("seed: created %d common medicines", len(items))
}
