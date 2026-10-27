package seed

import "github.com/neko233-com/server-medical-system/internal/models"

// commonMedicines returns the seed formulary of common drugs. The admin panel
// can extend this list at runtime; this is a representative starting set.
func commonMedicines() []models.Medicine {
	active := true
	mk := func(code, zh, en, cat, catEN, spec, mfr, unit string, price float64) models.Medicine {
		return models.Medicine{
			Code: code, NameZH: zh, NameEN: en, Category: cat, CategoryEN: catEN,
			Spec: spec, Manufacturer: mfr, Unit: unit, Price: price, Currency: "CNY", IsActive: active,
		}
	}
	return []models.Medicine{
		// 抗生素 Antibiotics
		mk("AB001", "阿莫西林胶囊", "Amoxicillin Capsule", "抗生素", "Antibiotics", "0.25g*24粒", "联邦制药", "盒", 15.0),
		mk("AB002", "头孢克肟片", "Cefixime Tablet", "抗生素", "Antibiotics", "0.1g*6片", "白云山", "盒", 22.5),
		mk("AB003", "阿奇霉素片", "Azithromycin Tablet", "抗生素", "Antibiotics", "0.25g*6片", "辉瑞", "盒", 38.0),
		mk("AB004", "左氧氟沙星片", "Levofloxacin Tablet", "抗生素", "Antibiotics", "0.5g*7片", "第一三共", "盒", 28.0),
		mk("AB005", "克林霉素磷酸酯注射液", "Clindamycin Injection", "抗生素", "Antibiotics", "2ml:0.3g", "国药集团", "支", 6.5),
		mk("AB006", "甲硝唑片", "Metronidazole Tablet", "抗生素", "Antibiotics", "0.2g*21片", "同仁堂", "盒", 9.0),

		// 解热镇痛 Antipyretic-Analgesic
		mk("AP001", "布洛芬缓释胶囊", "Ibuprofen Sustained-Release Capsule", "解热镇痛", "Antipyretic-Analgesic", "0.3g*20粒", "中美天津史克", "盒", 18.0),
		mk("AP002", "对乙酰氨基酚片", "Paracetamol Tablet", "解热镇痛", "Antipyretic-Analgesic", "0.5g*10片", "必理通", "盒", 12.0),
		mk("AP003", "阿司匹林肠溶片", "Aspirin Enteric-Coated Tablet", "解热镇痛", "Antipyretic-Analgesic", "100mg*30片", "拜耳", "盒", 16.5),
		mk("AP004", "双氯芬酸钠缓释片", "Diclofenac Sodium Sustained-Release Tablet", "解热镇痛", "Antipyretic-Analgesic", "75mg*10片", "扶他林", "盒", 21.0),
		mk("AP005", "复方氨林巴比妥注射液", "Compound Aminopyrine Injection", "解热镇痛", "Antipyretic-Analgesic", "2ml", "哈药集团", "支", 3.5),

		// 心血管 Cardiovascular
		mk("CV001", "硝苯地平控释片", "Nifedipine Controlled-Release Tablet", "心血管", "Cardiovascular", "30mg*7片", "拜新同", "盒", 32.0),
		mk("CV002", "缬沙坦胶囊", "Valsartan Capsule", "心血管", "Cardiovascular", "80mg*7粒", "代文", "盒", 35.0),
		mk("CV003", "美托洛尔片", "Metoprolol Tartrate Tablet", "心血管", "Cardiovascular", "25mg*20片", "倍他乐克", "盒", 19.0),
		mk("CV004", "阿托伐他汀钙片", "Atorvastatin Calcium Tablet", "心血管", "Cardiovascular", "20mg*7片", "立普妥", "盒", 42.0),
		mk("CV005", "硝酸甘油片", "Nitroglycerin Tablet", "心血管", "Cardiovascular", "0.5mg*100片", "北京益民", "瓶", 25.0),
		mk("CV006", "单硝酸异山梨酯片", "Isosorbide Mononitrate Tablet", "心血管", "Cardiovascular", "20mg*48片", "鲁南制药", "盒", 27.0),

		// 消化系统 Digestive
		mk("DG001", "奥美拉唑肠溶胶囊", "Omeprazole Enteric Capsule", "消化系统", "Digestive", "20mg*14粒", "洛赛克", "盒", 36.0),
		mk("DG002", "雷贝拉唑钠肠溶片", "Rabeprazole Sodium Enteric Tablet", "消化系统", "Digestive", "10mg*7片", "波利特", "盒", 45.0),
		mk("DG003", "多潘立酮片", "Domperidone Tablet", "消化系统", "Digestive", "10mg*30片", "吗丁啉", "盒", 23.0),
		mk("DG004", "蒙脱石散", "Smectite Powder", "消化系统", "Digestive", "3g*10袋", "思密达", "盒", 26.0),
		mk("DG005", "复方甘草片", "Compound Licorice Tablet", "消化系统", "Digestive", "100片", "太极集团", "瓶", 8.0),
		mk("DG006", "乳果糖口服溶液", "Lactulose Oral Solution", "消化系统", "Digestive", "100ml", "杜密克", "瓶", 33.0),

		// 呼吸系统 Respiratory
		mk("RS001", "氨溴索口服溶液", "Ambroxol Oral Solution", "呼吸系统", "Respiratory", "100ml", "沐舒坦", "瓶", 24.0),
		mk("RS002", "异丙托溴铵气雾剂", "Ipratropium Aerosol", "呼吸系统", "Respiratory", "10ml", "爱全乐", "瓶", 48.0),
		mk("RS003", "孟鲁司特钠咀嚼片", "Montelukast Sodium Chewable Tablet", "呼吸系统", "Respiratory", "5mg*5片", "顺尔宁", "盒", 39.0),
		mk("RS004", "布地奈德福莫特罗粉吸入剂", "Budesonide/Formoterol Inhaler", "呼吸系统", "Respiratory", "160/4.5μg*60吸", "信必可", "盒", 198.0),

		// 内分泌 Endocrine
		mk("EN001", "二甲双胍缓释片", "Metformin Sustained-Release Tablet", "内分泌", "Endocrine", "0.5g*30片", "格华止", "盒", 28.0),
		mk("EN002", "阿卡波糖片", "Acarbose Tablet", "内分泌", "Endocrine", "50mg*30片", "拜糖平", "盒", 52.0),
		mk("EN003", "重组人胰岛素注射液", "Recombinant Human Insulin Injection", "内分泌", "Endocrine", "300IU", "诺和灵", "支", 65.0),
		mk("EN004", "左甲状腺素钠片", "Levothyroxine Sodium Tablet", "内分泌", "Endocrine", "50μg*100片", "优甲乐", "盒", 30.0),

		// 神经系统 Nervous System
		mk("NS001", "地西泮片", "Diazepam Tablet", "神经系统", "Nervous System", "2.5mg*20片", "安定", "盒", 11.0),
		mk("NS002", "艾司唑仑片", "Estazolam Tablet", "神经系统", "Nervous System", "1mg*20片", "舒乐安定", "盒", 13.0),
		mk("NS003", "甲钴胺片", "Mecobalamin Tablet", "神经系统", "Nervous System", "0.5mg*48片", "弥可保", "盒", 34.0),
		mk("NS004", "胞磷胆碱钠胶囊", "Citicoline Sodium Capsule", "神经系统", "Nervous System", "0.1g*24粒", "星瑞灵", "盒", 29.0),

		// 抗过敏 Antiallergic
		mk("AL001", "氯雷他定片", "Loratadine Tablet", "抗过敏", "Antiallergic", "10mg*6片", "开瑞坦", "盒", 19.0),
		mk("AL002", "盐酸西替利嗪片", "Cetirizine Hydrochloride Tablet", "抗过敏", "Antiallergic", "10mg*10片", "仙特明", "盒", 17.0),
		mk("AL003", "地塞米松注射液", "Dexamethasone Injection", "抗过敏", "Antiallergic", "1ml:5mg", "天津金耀", "支", 2.5),

		// 维生素与营养 Vitamins & Nutrition
		mk("VN001", "维生素C片", "Vitamin C Tablet", "维生素与营养", "Vitamins & Nutrition", "100mg*100片", "力度伸", "瓶", 15.0),
		mk("VN002", "复合维生素B片", "Vitamin B Complex Tablet", "维生素与营养", "Vitamins & Nutrition", "100片", "同仁堂", "瓶", 12.0),
		mk("VN003", "碳酸钙D3片", "Calcium Carbonate D3 Tablet", "维生素与营养", "Vitamins & Nutrition", "0.6g*60片", "钙尔奇", "盒", 38.0),
		mk("VN004", "葡萄糖酸锌口服溶液", "Zinc Gluconate Oral Solution", "维生素与营养", "Vitamins & Nutrition", "10ml*10支", "哈药集团", "盒", 22.0),

		// 外用 Topical
		mk("TP001", "红霉素软膏", "Erythromycin Ointment", "外用", "Topical", "10g", "马应龙", "支", 6.0),
		mk("TP002", "莫匹罗星软膏", "Mupirocin Ointment", "外用", "Topical", "5g", "百多邦", "支", 18.0),
		mk("TP003", "炉甘石洗剂", "Calamine Lotion", "外用", "Topical", "100ml", "信龙", "瓶", 9.0),
		mk("TP004", "云南白药气雾剂", "Yunnan Baiyao Spray", "外用", "Topical", "85g+30g", "云南白药", "盒", 39.0),
		mk("TP005", "过氧化氢溶液", "Hydrogen Peroxide Solution", "外用", "Topical", "100ml", "利尔康", "瓶", 4.0),

		// 中成药 TCM
		mk("TC001", "连花清瘟胶囊", "Lianhua Qingwen Capsule", "中成药", "TCM", "0.35g*24粒", "以岭药业", "盒", 26.0),
		mk("TC002", "感冒灵颗粒", "Ganmaoling Granules", "中成药", "TCM", "10g*9袋", "999", "盒", 13.0),
		mk("TC003", "板蓝根颗粒", "Banlangen Granules", "中成药", "TCM", "10g*20袋", "白云山", "盒", 16.0),
		mk("TC004", "藿香正气口服液", "Huoxiang Zhengqi Oral Liquid", "中成药", "TCM", "10ml*10支", "太极集团", "盒", 19.0),
		mk("TC005", "复方丹参滴丸", "Compound Danshen Dripping Pills", "中成药", "TCM", "27mg*180丸", "天士力", "盒", 29.0),
		mk("TC006", "血塞通软胶囊", "Xuesaitong Soft Capsule", "中成药", "TCM", "0.33g*20粒", "昆药集团", "盒", 31.0),

		// 五官科 ENT
		mk("EN010", "左氧氟沙星滴眼液", "Levofloxacin Eye Drops", "五官科", "ENT", "5ml", "可乐必妥", "支", 16.0),
		mk("EN011", "氯霉素滴眼液", "Chloramphenicol Eye Drops", "五官科", "ENT", "8ml", "润舒", "支", 5.0),

		// 其他 Others
		mk("OT001", "生理盐水", "Sodium Chloride 0.9%", "其他", "Others", "500ml", "科伦药业", "瓶", 3.0),
		mk("OT002", "葡萄糖注射液", "Glucose 5% Injection", "其他", "Others", "500ml", "科伦药业", "瓶", 3.5),
		mk("OT003", "破伤风抗毒素", "Tetanus Antitoxin", "其他", "Others", "1500IU", "生物制品研究所", "支", 18.0),
		mk("OT004", "口服补液盐", "Oral Rehydration Salts", "其他", "Others", "13.95g*6袋", "博叶", "盒", 14.0),
		mk("OT005", "碘伏消毒液", "Povidone Iodine Solution", "其他", "Others", "100ml", "利尔康", "瓶", 5.0),
		mk("OT006", "创可贴", "Adhesive Bandage", "其他", "Others", "100片", "云南白药", "盒", 10.0),
	}
}
