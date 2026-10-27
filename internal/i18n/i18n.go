package i18n

import (
	"github.com/gin-gonic/gin"
)

// Messages maps a message key to its localized string.
type Messages map[string]string

// dict holds the supported locales. Add a new map here and register the code in
// detectLang to add another language — the frontend and backend stay in sync.
var dict = map[string]Messages{
	"zh-CN": {
		"invalid_request":           "请求格式错误",
		"invalid_credentials":       "用户名或密码错误",
		"unauthorized":              "未授权，请先登录",
		"forbidden":                 "权限不足",
		"not_found":                 "资源不存在",
		"server_error":              "服务器内部错误",
		"validation_error":          "数据校验失败",
		"name_required":             "姓名不能为空",
		"username_required":         "用户名不能为空",
		"password_required":         "密码不能为空",
		"medicine_code_exists":      "药品编码已存在",
		"patient_not_found":         "患者不存在",
		"case_not_found":            "病例不存在",
		"medicine_not_found":        "药品不存在",
		"prescription_created":      "处方已开具",
		"created":                   "创建成功",
		"updated":                   "更新成功",
		"deleted":                   "删除成功",
		"lang_saved":                "语言偏好已保存",
		"role_not_found":            "角色不存在",
		"role_code_exists":          "角色编码已存在",
		"user_not_found":            "用户不存在",
		"username_exists":           "用户名已存在",
		"cannot_delete_system_role": "系统角色不可删除",
		"cannot_disable_self":       "不能停用当前登录账号",
		"room_not_found":            "房间不存在",
		"printer_not_found":         "打印机不存在",
		"surgery_not_found":         "手术记录不存在",
		"study_not_found":           "影像检查不存在",
		"image_not_found":           "影像图片不存在",
		"invalid_status":            "非法的状态值",
		"file_too_large":            "文件过大",
		"unsupported_file_type":     "不支持的文件类型",
		"no_target_printer":         "未找到可用打印机",
		"binding_conflict":          "绑定冲突",
	},
	"en-US": {
		"invalid_request":           "Invalid request format",
		"invalid_credentials":       "Incorrect username or password",
		"unauthorized":              "Unauthorized, please sign in",
		"forbidden":                 "Insufficient permissions",
		"not_found":                 "Resource not found",
		"server_error":              "Internal server error",
		"validation_error":          "Validation failed",
		"name_required":             "Name is required",
		"username_required":         "Username is required",
		"password_required":         "Password is required",
		"medicine_code_exists":      "Medicine code already exists",
		"patient_not_found":         "Patient not found",
		"case_not_found":            "Medical case not found",
		"medicine_not_found":        "Medicine not found",
		"prescription_created":      "Prescription created",
		"created":                   "Created",
		"updated":                   "Updated",
		"deleted":                   "Deleted",
		"lang_saved":                "Language preference saved",
		"role_not_found":            "Role not found",
		"role_code_exists":          "Role code already exists",
		"user_not_found":            "User not found",
		"username_exists":           "Username already exists",
		"cannot_delete_system_role": "System role cannot be deleted",
		"cannot_disable_self":       "Cannot disable your own account",
		"room_not_found":            "Room not found",
		"printer_not_found":         "Printer not found",
		"surgery_not_found":         "Surgery record not found",
		"study_not_found":           "Imaging study not found",
		"image_not_found":           "Imaging image not found",
		"invalid_status":            "Invalid status value",
		"file_too_large":            "File too large",
		"unsupported_file_type":     "Unsupported file type",
		"no_target_printer":         "No available printer found",
		"binding_conflict":          "Binding conflict",
	},
}

// SupportedLangs lists locale codes the API understands.
var SupportedLangs = []string{"zh-CN", "en-US"}

// detectLang resolves a request locale from the `lang` query param or the
// Accept-Language header, falling back to zh-CN.
func detectLang(c *gin.Context) string {
	lang := c.Query("lang")
	if lang == "" {
		lang = c.GetHeader("Accept-Language")
	}
	switch {
	case lang == "zh" || lang == "zh-CN" || lang == "zh_CN":
		return "zh-CN"
	case lang == "en" || lang == "en-US" || lang == "en_US":
		return "en-US"
	default:
		return "zh-CN"
	}
}

// Middleware stores the resolved language on the request context.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("lang", detectLang(c))
		c.Next()
	}
}

// T translates a key for the current request's language, with en-US fallback.
func T(c *gin.Context, key string) string {
	if lang, ok := c.Get("lang"); ok {
		if l, ok := lang.(string); ok {
			if m, ok := dict[l]; ok {
				if v, ok := m[key]; ok {
					return v
				}
			}
		}
	}
	if m, ok := dict["en-US"]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}
