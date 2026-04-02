import React, { useState, useEffect } from "react";
import {
  GetServerURL,
  SetServerURL,
  CheckSoftEtherInstalled,
  GetSoftEtherVersion,
  CheckVPNReady,
} from "../api";

const pageStyle = {
  maxWidth: "600px",
  margin: "0 auto",
  display: "flex",
  flexDirection: "column",
  gap: "16px",
  animation: "fadeIn 0.3s ease-out",
  direction: "rtl",
  textAlign: "right",
};

const card = {
  background: "#1a1a35",
  borderRadius: "12px",
  padding: "24px",
  border: "1px solid #2a2a45",
};

const sectionTitle = {
  fontSize: "0.82rem",
  fontWeight: 600,
  marginBottom: "16px",
  color: "#8888aa",
  letterSpacing: "0.5px",
};

const infoRow = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "10px 0",
  borderBottom: "1px solid #1a1a30",
};

const labelText = { color: "#8888aa", fontSize: "0.9rem" };
const valueText = { color: "#e8e8f0", fontWeight: 500, fontSize: "0.9rem" };

const inputStyle = {
  width: "100%",
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.95rem",
  outline: "none",
  transition: "border-color 0.2s ease",
  direction: "ltr",
  textAlign: "left",
  fontFamily: "monospace",
};

const btnPrimary = {
  padding: "10px 20px",
  borderRadius: "10px",
  border: "none",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.88rem",
  flexShrink: 0,
  fontFamily: "'Vazirmatn', sans-serif",
};

const statusDot = (ok) => ({
  width: "10px",
  height: "10px",
  borderRadius: "50%",
  background: ok ? "#00e676" : "#ff5252",
  boxShadow: `0 0 6px ${ok ? "#00e67666" : "#ff525266"}`,
  display: "inline-block",
  marginLeft: "8px",
});

const statusTag = (ok) => ({
  display: "inline-flex",
  alignItems: "center",
  padding: "4px 12px",
  borderRadius: "8px",
  fontSize: "0.82rem",
  fontWeight: 500,
  background: ok ? "#00e67612" : "#ff525212",
  color: ok ? "#00e676" : "#ff5252",
  border: `1px solid ${ok ? "#00e67622" : "#ff525222"}`,
});

const successMsg = {
  color: "#00e676",
  fontSize: "0.82rem",
  marginTop: "8px",
};

const offlineBanner = {
  background: "#ff525215",
  border: "1px solid #ff525233",
  borderRadius: "10px",
  padding: "12px 16px",
  fontSize: "0.85rem",
  color: "#ff5252",
  lineHeight: 1.5,
  textAlign: "center",
};

export default function SettingsPage() {
  const [serverURL, setServerURL] = useState("");
  const [saved, setSaved] = useState(false);
  const [seInstalled, setSeInstalled] = useState(null);
  const [seVersion, setSeVersion] = useState("--");
  const [vpnReady, setVpnReady] = useState(null);
  const [offline, setOffline] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const url = await GetServerURL();
        setServerURL(url);
        setOffline(false);
      } catch (err) {
        const errStr = String(err);
        if (errStr.includes("fetch") || errStr.includes("network") || errStr.includes("Failed") || errStr.includes("ECONNREFUSED")) {
          setOffline(true);
        }
      }
      try {
        const installed = await CheckSoftEtherInstalled();
        setSeInstalled(installed);
      } catch {
        setSeInstalled(false);
      }
      try {
        const ver = await GetSoftEtherVersion();
        setSeVersion(ver);
      } catch {
        setSeVersion("نصب نشده");
      }
      try {
        const ready = await CheckVPNReady();
        setVpnReady(ready);
      } catch {
        setVpnReady(null);
      }
    })();
  }, []);

  const handleSaveServer = async () => {
    try {
      await SetServerURL(serverURL);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch {
      // ignore
    }
  };

  return (
    <div style={pageStyle}>
      {offline && (
        <div style={offlineBanner}>
          سرور در دسترس نیست. تلگرام: @coded_pro
        </div>
      )}

      {/* Server config */}
      <div style={card}>
        <div style={sectionTitle}>تنظیمات سرور</div>
        <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
          <input
            style={{ ...inputStyle, flex: 1 }}
            placeholder="http://server:8080"
            value={serverURL}
            onChange={(e) => {
              setServerURL(e.target.value);
              setSaved(false);
            }}
            onFocus={(e) => { e.target.style.borderColor = "#7c4dff66"; }}
            onBlur={(e) => { e.target.style.borderColor = "#2a2a45"; }}
          />
          <button style={btnPrimary} onClick={handleSaveServer}>
            ذخیره
          </button>
        </div>
        {saved && <div style={successMsg}>آدرس سرور با موفقیت ذخیره شد</div>}
      </div>

      {/* SoftEther status */}
      <div style={card}>
        <div style={sectionTitle}>کلاینت SoftEther VPN</div>
        <div style={infoRow}>
          <span style={labelText}>نصب شده</span>
          <span style={statusTag(seInstalled)}>
            <span style={statusDot(seInstalled)} />
            {seInstalled === null ? "در حال بررسی..." : seInstalled ? "بله" : "یافت نشد"}
          </span>
        </div>
        <div style={infoRow}>
          <span style={labelText}>نسخه</span>
          <span style={{ ...valueText, fontFamily: "monospace", direction: "ltr" }}>{seVersion}</span>
        </div>
        {vpnReady && (
          <>
            <div style={infoRow}>
              <span style={labelText}>دستور VPN</span>
              <span style={statusTag(vpnReady.vpncmd_found)}>
                <span style={statusDot(vpnReady.vpncmd_found)} />
                {vpnReady.vpncmd_found ? "یافت شد" : "یافت نشد"}
              </span>
            </div>
            <div style={infoRow}>
              <span style={labelText}>سرویس فعال</span>
              <span style={statusTag(vpnReady.service_running)}>
                <span style={statusDot(vpnReady.service_running)} />
                {vpnReady.service_running ? "در حال اجرا" : "متوقف"}
              </span>
            </div>
            <div style={{ ...infoRow, borderBottom: "none" }}>
              <span style={labelText}>آماده</span>
              <span style={statusTag(vpnReady.ready)}>
                <span style={statusDot(vpnReady.ready)} />
                {vpnReady.ready ? "آماده" : vpnReady.message || "آماده نیست"}
              </span>
            </div>
          </>
        )}
        {!seInstalled && seInstalled !== null && (
          <div
            style={{
              marginTop: "12px",
              padding: "12px 16px",
              background: "#ffab0012",
              borderRadius: "8px",
              border: "1px solid #ffab0022",
              fontSize: "0.85rem",
              color: "#ffab00",
              lineHeight: 1.5,
            }}
          >
            کلاینت SoftEther VPN برای عملکرد VPN لازم است.
            لطفا آن را از وبسایت رسمی SoftEther دانلود و نصب کنید.
          </div>
        )}
      </div>

      {/* VPN Preferences */}
      <div style={card}>
        <div style={sectionTitle}>تنظیمات VPN</div>
        <div style={infoRow}>
          <span style={labelText}>حالت پروتکل</span>
          <span style={{ ...valueText, direction: "ltr" }}>TCP (Multi-stream)</span>
        </div>
        <div style={infoRow}>
          <span style={labelText}>اتصالات TCP</span>
          <span style={{ ...valueText, fontFamily: "monospace", direction: "ltr" }}>8</span>
        </div>
        <div style={{ ...infoRow, borderBottom: "none" }}>
          <span style={labelText}>شتاب‌دهنده UDP</span>
          <span style={statusTag(true)}>
            <span style={statusDot(true)} />
            فعال
          </span>
        </div>
        <div
          style={{
            marginTop: "12px",
            padding: "10px 14px",
            background: "#7c4dff08",
            borderRadius: "8px",
            fontSize: "0.82rem",
            color: "#8888aa",
            lineHeight: 1.5,
          }}
        >
          VPN از 8 اتصال همزمان TCP روی پورت 443 برای حداکثر پایداری استفاده می‌کند.
          شتاب‌دهنده UDP برای تاخیر کمتر در صورت امکان فعال است.
        </div>
      </div>

      {/* About */}
      <div style={card}>
        <div style={sectionTitle}>درباره</div>
        <div style={infoRow}>
          <span style={labelText}>برنامه</span>
          <span style={valueText}>Dotachi</span>
        </div>
        <div style={infoRow}>
          <span style={labelText}>نسخه</span>
          <span style={{ ...valueText, fontFamily: "monospace", direction: "ltr" }}>0.1.0</span>
        </div>
        <div style={{ ...infoRow, borderBottom: "none" }}>
          <span style={labelText}>پلتفرم</span>
          <span style={valueText}>Wails Desktop</span>
        </div>
      </div>
    </div>
  );
}
