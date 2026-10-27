"""End-to-end smoke test for the new hospital modules.

Covers: appointments (预约挂号), beds/wards (住院床位), billing (收费结算) and the
extended operations analytics. Run against a freshly seeded server.
"""
import json
import os
import sys
import urllib.request
import urllib.error
import uuid

BASE = "http://127.0.0.1:6970"

failures = []


def check(name, cond, detail=""):
    if cond:
        print(f"  PASS  {name}")
    else:
        print(f"  FAIL  {name} :: {detail}")
        failures.append(name)


def req(method, path, token=None, data=None, raw=None, ctype="application/json"):
    url = BASE + path
    headers = {}
    if token:
        headers["Authorization"] = "Bearer " + token
    body = None
    if raw is not None:
        body = raw
        headers["Content-Type"] = ctype
    elif data is not None:
        body = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            return resp.status, resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace")
    except Exception as e:
        return 0, repr(e)


def login(user, pw):
    s, b = req("POST", "/api/auth/login", data={"username": user, "password": pw})
    if s != 200:
        raise SystemExit(f"login {user} failed: {s} {b}")
    return json.loads(b)["token"]


def j(b):
    try:
        return json.loads(b)
    except Exception:
        return {}


print("== auth ==")
root = login("root", "root")
doctor = login("doctor", "doctor123")
recept = login("reception", "recept123")
nurse = login("nurse", "nurse123")
check("4 roles logged in", all([root, doctor, recept, nurse]))

print("== seed data ==")
s, b = req("GET", "/api/wards", token=root)
wards = j(b).get("items", [])
check("wards seeded", s == 200 and len(wards) >= 3, f"status={s} n={len(wards)}")
s, b = req("GET", "/api/beds", token=root, data=None)
allbeds = j(b).get("items", [])
check("beds seeded", s == 200 and len(allbeds) >= 20, f"status={s} n={len(allbeds)}")

print("== patient prerequisite ==")
pid = str(uuid.uuid4())[:8]
s, b = req("POST", "/api/patients", token=root, data={
    "name": "测试患者CLINIC", "gender": "male", "age": 45,
    "phone": "13900000000", "code": "CLINIC-" + pid,
})
if s not in (200, 201):
    raise SystemExit(f"patient create failed: {s} {b}")
patient_id = j(b)["item"]["id"]
check("patient created", patient_id > 0)

print("== appointments ==")
s, b = req("POST", "/api/appointments", token=recept, data={
    "patient_id": patient_id, "doctor_id": 2, "department": "内科",
    "source": "online", "appt_type": "general", "reason": "复诊",
    "slot_at": "2026-10-28T09:30:00+08:00",
})
appt = j(b).get("item", {})
check("appointment created", s in (200, 201) and appt.get("appt_no", "").startswith("APT"),
      f"status={s} body={b[:160]}")
appt_id = appt.get("id", 0)
check("appointment patient name resolved", appt.get("patient_name") == "测试患者CLINIC",
      str(appt.get("patient_name")))

s, b = req("GET", f"/api/appointments/{appt_id}", token=doctor)
check("appointment readable", s == 200 and j(b).get("id") == appt_id, f"status={s}")

s, b = req("PUT", f"/api/appointments/{appt_id}/status", token=recept, data={"status": "arrived"})
check("appointment -> arrived", s == 200 and j(b).get("item", {}).get("status") == "arrived", f"status={s}")

s, b = req("PUT", f"/api/appointments/{appt_id}/status", token=recept, data={"status": "completed"})
check("appointment -> completed", s == 200 and j(b).get("item", {}).get("status") == "completed", f"status={s}")

s, b = req("PUT", f"/api/appointments/{appt_id}/status", token=recept, data={"status": "bogus"})
check("invalid appointment status rejected", s == 400, f"status={s}")

# RBAC: doctor has appointment:read + manage? doctor role grants appointment:read only for now.
s, b = req("DELETE", f"/api/appointments/{appt_id}", token=doctor)
check("doctor blocked from appointment delete (RBAC)", s == 403, f"status={s}")

s, b = req("GET", "/api/appointments", token=doctor)
check("doctor can list appointments", s == 200, f"status={s}")

print("== beds / wards ==")
s, b = req("GET", "/api/beds", token=nurse)
beds = j(b).get("items", [])
free_bed = next((x for x in beds if x["status"] == "available"), None)
occupied_bed = next((x for x in beds if x["status"] == "occupied"), None)
check("bed list readable by nurse", s == 200 and len(beds) > 0, f"status={s}")
check("seed has both free and occupied beds", free_bed is not None and occupied_bed is not None)

if free_bed:
    s, b = req("PUT", f"/api/beds/{free_bed['id']}/admit", token=nurse, data={"patient_id": patient_id})
    ad = j(b).get("item", {})
    check("bed admit", s == 200 and ad.get("status") == "occupied", f"status={s} {b[:140]}")
    check("bed admit records admitted_at", ad.get("admitted_at") is not None)
    s, b = req("PUT", f"/api/beds/{free_bed['id']}/admit", token=nurse, data={"patient_id": patient_id})
    check("double admit rejected", s == 409, f"status={s}")
    s, b = req("PUT", f"/api/beds/{free_bed['id']}/discharge", token=nurse)
    dis = j(b).get("item", {})
    check("bed discharge releases bed", s == 200 and dis.get("status") == "available", f"status={s} {b[:140]}")
    check("discharge clears patient", not dis.get("patient_id"), str(dis.get("patient_id")))

s, b = req("POST", "/api/wards", token=root, data={
    "code": "W-TEST", "name_zh": "测试病区", "name_en": "Test Ward",
    "department": "测试科", "floor": "9F",
})
w = j(b).get("item", {})
check("ward created", s in (200, 201) and w.get("id", 0) > 0, f"status={s} {b[:140]}")
ward_id = w.get("id", 0)

s, b = req("POST", "/api/beds", token=root, data={
    "ward_id": ward_id, "bed_no": "01", "daily_rate": 120.0,
})
nb = j(b).get("item", {})
check("bed created in ward", s in (200, 201) and nb.get("code", "").startswith("W-TEST"),
      f"status={s} {b[:140]}")

s, b = req("DELETE", f"/api/wards/{ward_id}", token=root)
check("ward with beds cannot be deleted (409)", s == 409, f"status={s} {b[:120]}")

s, b = req("DELETE", f"/api/beds/{nb.get('id', 0)}", token=root)
check("empty bed deleted", s == 200, f"status={s}")
s, b = req("DELETE", f"/api/wards/{ward_id}", token=root)
check("empty ward deleted", s == 200, f"status={s}")

print("== billing ==")
s, b = req("POST", "/api/invoices", token=recept, data={
    "patient_id": patient_id, "department": "内科", "doctor_id": 2, "currency": "CNY",
})
inv = j(b).get("item", {})
check("invoice created", s in (200, 201) and inv.get("invoice_no", "").startswith("INV"),
      f"status={s} {b[:140]}")
inv_id = inv.get("id", 0)
check("invoice starts pending", inv.get("status") == "pending", str(inv.get("status")))

s, b = req("POST", f"/api/invoices/{inv_id}/items", token=recept, data={
    "item_type": "consult", "name_zh": "门诊诊查费", "name_en": "Consultation",
    "qty": 1, "unit": "次", "unit_price": 50.0,
})
check("invoice item added", s in (200, 201) and j(b).get("invoice", {}).get("subtotal") == 50.0,
      f"status={s} {b[:160]}")

s, b = req("POST", f"/api/invoices/{inv_id}/items", token=recept, data={
    "item_type": "medicine", "name_zh": "阿莫西林胶囊", "name_en": "Amoxicillin",
    "qty": 2, "unit": "盒", "unit_price": 12.5,
})
check("invoice total recomputed", s in (200, 201) and j(b).get("invoice", {}).get("total") == 75.0,
      f"status={s} {b[:160]}")

s, b = req("POST", f"/api/invoices/{inv_id}/pay", token=recept, data={"payment_method": "wechat"})
pd = j(b).get("item", {})
check("invoice paid", s == 200 and pd.get("status") == "paid", f"status={s} {b[:140]}")
check("invoice payment method stored", pd.get("payment_method") == "wechat", str(pd.get("payment_method")))
check("invoice paid_at set", pd.get("paid_at") is not None)

s, b = req("POST", f"/api/invoices/{inv_id}/pay", token=recept, data={"payment_method": "cash"})
check("double pay rejected", s == 409, f"status={s}")

s, b = req("DELETE", f"/api/invoices/{inv_id}", token=recept)
check("paid invoice cannot be deleted (409)", s == 409, f"status={s} {b[:120]}")

s, b = req("POST", f"/api/invoices/{inv_id}/refund", token=recept)
check("invoice refunded", s == 200 and j(b).get("item", {}).get("status") == "refunded", f"status={s}")

# RBAC negative: doctor lacks billing:manage
s, b = req("POST", "/api/invoices", token=doctor, data={"patient_id": patient_id})
check("doctor blocked from invoice create (RBAC)", s == 403, f"status={s}")

print("== operations analytics ==")
s, b = req("GET", "/api/operations/overview", token=root)
ov = j(b)
check("overview returns bed occupancy", s == 200 and "bed_occupancy" in ov, f"status={s} keys={list(ov)[:8]}")
check("overview returns paid revenue", "paid_month" in ov and "net_revenue_month" in ov)
check("overview returns appointments", "appointments_today" in ov)
check("overview beds_total > 0", ov.get("beds_total", 0) > 0, str(ov.get("beds_total")))

s, b = req("GET", "/api/operations/status", token=root)
st = j(b)
check("status includes beds by ward", s == 200 and isinstance(st.get("beds"), list) and len(st["beds"]) > 0,
      f"status={s}")

s, b = req("GET", "/api/operations/revenue", token=root)
rv = j(b)
check("revenue breakdown", s == 200 and "by_status" in rv and "by_method" in rv and "by_item" in rv,
      f"status={s} keys={list(rv)}")
check("revenue total_paid numeric", isinstance(rv.get("total_paid"), (int, float)), str(rv.get("total_paid")))

# RBAC: doctor lacks report:read
s, b = req("GET", "/api/operations/overview", token=doctor)
check("doctor blocked from operations (RBAC)", s == 403, f"status={s}")

print()
if failures:
    print(f"FAILED {len(failures)}: {failures}")
    sys.exit(1)
print("ALL CLINIC MODULE TESTS PASSED")
