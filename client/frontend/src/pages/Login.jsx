import React, { useState, useEffect } from "react";
import {
  Login as doLogin,
  Register,
  GetServerURL,
  SetServerURL,
  CheckSoftEtherInstalled,
} from "../api";

const wrapperStyle = {
  display: "flex",
  justifyContent: "center",
  alignItems: "center",
  minHeight: "100vh",
  background: "linear-gradient(135deg, #0a0a1a 0%, #0f0f2a 50%, #0a0a1a 100%)",
  position: "relative",
  overflow: "hidden",
  direction: "rtl",
};

// Subtle background grid pattern via pseudo-element simulation
const bgOverlay = {
  position: "absolute",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background:
    "radial-gradient(ellipse at 50% 0%, #7c4dff08 0%, transparent 60%), " +
    "radial-gradient(ellipse at 80% 100%, #7c4dff05 0%, transparent 40%)",
  pointerEvents: "none",
};

const cardStyle = {
  background: "#141428",
  borderRadius: "16px",
  padding: "48px 40px",
  width: "100%",
  maxWidth: "420px",
  boxShadow: "0 8px 40px rgba(0, 0, 0, 0.5), 0 0 60px rgba(124, 77, 255, 0.08)",
  border: "1px solid #2a2a45",
  position: "relative",
  zIndex: 1,
  animation: "fadeIn 0.4s ease-out",
  direction: "rtl",
  textAlign: "right",
};

const logoStyle = {
  fontSize: "2.4rem",
  fontWeight: 800,
  color: "#7c4dff",
  textAlign: "center",
  marginBottom: "4px",
  letterSpacing: "4px",
  textShadow: "0 0 30px rgba(124, 77, 255, 0.3)",
};

const subtitleStyle = {
  textAlign: "center",
  color: "#8888aa",
  marginBottom: "32px",
  fontSize: "0.9rem",
  letterSpacing: "1px",
  fontWeight: 500,
};

const inputStyle = {
  width: "100%",
  padding: "12px 16px",
  marginBottom: "14px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.95rem",
  outline: "none",
  transition: "border-color 0.2s ease, box-shadow 0.2s ease",
  direction: "rtl",
  textAlign: "right",
  fontFamily: "'Vazirmatn', sans-serif",
};

const inputFocusHandler = (e) => {
  e.target.style.borderColor = "#7c4dff66";
  e.target.style.boxShadow = "0 0 0 2px #7c4dff22";
};

const inputBlurHandler = (e) => {
  e.target.style.borderColor = "#2a2a45";
  e.target.style.boxShadow = "none";
};

const btnPrimary = {
  width: "100%",
  padding: "14px",
  borderRadius: "10px",
  border: "none",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  color: "#fff",
  fontSize: "1rem",
  fontWeight: 600,
  cursor: "pointer",
  marginBottom: "14px",
  transition: "all 0.2s ease",
  letterSpacing: "0.5px",
  boxShadow: "0 4px 16px rgba(124, 77, 255, 0.25)",
  fontFamily: "'Vazirmatn', sans-serif",
};

const errorStyle = {
  color: "#ff5252",
  fontSize: "0.85rem",
  marginBottom: "14px",
  textAlign: "center",
  padding: "10px 14px",
  background: "#ff525212",
  borderRadius: "8px",
  border: "1px solid #ff525222",
};

const warningBanner = {
  background: "#ffab0015",
  border: "1px solid #ffab0033",
  borderRadius: "10px",
  padding: "12px 16px",
  marginBottom: "20px",
  fontSize: "0.85rem",
  color: "#ffab00",
  lineHeight: 1.5,
  textAlign: "center",
};

const offlineBanner = {
  background: "#ff525215",
  border: "1px solid #ff525233",
  borderRadius: "10px",
  padding: "12px 16px",
  marginBottom: "20px",
  fontSize: "0.85rem",
  color: "#ff5252",
  lineHeight: 1.5,
  textAlign: "center",
};

const advancedToggle = {
  background: "none",
  border: "none",
  color: "#555577",
  cursor: "pointer",
  fontSize: "0.82rem",
  textAlign: "center",
  display: "block",
  width: "100%",
  padding: "8px 0",
  marginTop: "8px",
  transition: "color 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

const advancedSection = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px",
  marginBottom: "18px",
  border: "1px solid #2a2a45",
};

const serverRow = {
  display: "flex",
  gap: "8px",
};

const hintStyle = {
  fontSize: "0.75rem",
  color: "#666688",
  marginTop: "-8px",
  marginBottom: "14px",
  paddingRight: "4px",
  direction: "ltr",
  textAlign: "right",
};

const spinnerStyle = {
  display: "inline-block",
  width: "16px",
  height: "16px",
  border: "2px solid #ffffff44",
  borderTopColor: "#fff",
  borderRadius: "50%",
  animation: "spin 0.6s linear infinite",
  verticalAlign: "middle",
  marginLeft: "8px",
};

const tabRow = {
  display: "flex",
  marginBottom: "24px",
  borderRadius: "10px",
  overflow: "hidden",
  border: "1px solid #2a2a45",
};

const tabBtn = (active) => ({
  flex: 1,
  padding: "10px 0",
  background: active ? "#7c4dff" : "#0d0d20",
  color: active ? "#fff" : "#8888aa",
  border: "none",
  cursor: "pointer",
  fontSize: "0.9rem",
  fontWeight: 600,
  transition: "all 0.2s ease",
  letterSpacing: "0.3px",
  fontFamily: "'Vazirmatn', sans-serif",
});

const retryBtn = {
  background: "none",
  border: "1px solid #ff5252",
  color: "#ff5252",
  borderRadius: "8px",
  padding: "6px 16px",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 500,
  marginTop: "8px",
  fontFamily: "'Vazirmatn', sans-serif",
};

export default function LoginPage({ onLogin }) {
  const [isRegister, setIsRegister] = useState(false);
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [referralCode, setReferralCode] = useState("");
  const [serverURL, setServerURL] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [softEtherMissing, setSoftEtherMissing] = useState(false);
  const [offline, setOffline] = useState(false);

  // Check SoftEther status on mount
  useEffect(() => {
    (async () => {
      try {
        const installed = await CheckSoftEtherInstalled();
        if (!installed) {
          setSoftEtherMissing(true);
        }
      } catch {
        // If the call fails, assume not installed
        setSoftEtherMissing(true);
      }
    })();
  }, []);

  const handleToggleAdvanced = async () => {
    if (!showAdvanced) {
      try {
        const url = await GetServerURL();
        setServerURL(url);
      } catch {
        // ignore
      }
    }
    setShowAdvanced((v) => !v);
  };

  const handleSaveServer = async () => {
    try {
      await SetServerURL(serverURL);
    } catch {
      // ignore
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setOffline(false);
    setLoading(true);

    try {
      let result;
      if (isRegister) {
        if (!displayName.trim()) {
          setError("نام نمایشی الزامی است.");
          setLoading(false);
          return;
        }
        result = await Register(phone, password, displayName, referralCode);
      } else {
        result = await doLogin(phone, password);
      }
      onLogin(result);
    } catch (err) {
      const errStr = String(err);
      if (errStr.includes("fetch") || errStr.includes("network") || errStr.includes("Failed") || errStr.includes("ECONNREFUSED")) {
        setOffline(true);
      } else {
        setError(errStr);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={wrapperStyle}>
      <div style={bgOverlay} />
      <form style={cardStyle} onSubmit={handleSubmit}>
        <div style={logoStyle}>DOTACHI</div>
        <div style={subtitleStyle}>شبکه بازی لن</div>

        {offline && (
          <div style={offlineBanner}>
            سرور در دسترس نیست. تلگرام: @coded_pro
            <br />
            <button
              type="button"
              style={retryBtn}
              onClick={handleSubmit}
            >
              تلاش مجدد
            </button>
          </div>
        )}

        {softEtherMissing && (
          <div style={warningBanner}>
            کلاینت SoftEther VPN نصب نیست یا اجرا نشده است.
            <br />
            لطفا آن را نصب کنید تا بتوانید از قابلیت‌های VPN استفاده کنید.
          </div>
        )}

        {/* Login / Register tabs */}
        <div style={tabRow}>
          <button
            type="button"
            style={tabBtn(!isRegister)}
            onClick={() => {
              setIsRegister(false);
              setError("");
            }}
          >
            ورود
          </button>
          <button
            type="button"
            style={tabBtn(isRegister)}
            onClick={() => {
              setIsRegister(true);
              setError("");
            }}
          >
            ثبت‌نام
          </button>
        </div>

        <input
          style={inputStyle}
          type="tel"
          placeholder="شماره تلفن"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          onFocus={inputFocusHandler}
          onBlur={inputBlurHandler}
          required
        />
        <div style={hintStyle}>مثلا 09121234567</div>

        <input
          style={inputStyle}
          type="password"
          placeholder="رمز عبور"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onFocus={inputFocusHandler}
          onBlur={inputBlurHandler}
          required
        />

        {isRegister && (
          <>
            <input
              style={inputStyle}
              type="text"
              placeholder="نام نمایشی"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              onFocus={inputFocusHandler}
              onBlur={inputBlurHandler}
            />
            <input
              style={{ ...inputStyle, fontSize: "0.85rem" }}
              type="text"
              placeholder="کد معرف (اختیاری)"
              value={referralCode}
              onChange={(e) => setReferralCode(e.target.value)}
              onFocus={inputFocusHandler}
              onBlur={inputBlurHandler}
            />
          </>
        )}

        {error && <div style={errorStyle}>{error}</div>}

        <button
          style={{
            ...btnPrimary,
            opacity: loading ? 0.7 : 1,
            cursor: loading ? "not-allowed" : "pointer",
          }}
          type="submit"
          disabled={loading}
          onMouseEnter={(e) => {
            if (!loading) e.currentTarget.style.boxShadow = "0 6px 24px rgba(124, 77, 255, 0.4)";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.boxShadow = "0 4px 16px rgba(124, 77, 255, 0.25)";
          }}
        >
          {loading && <span style={spinnerStyle} />}
          {loading ? "لطفا صبر کنید..." : isRegister ? "ایجاد حساب" : "ورود"}
        </button>

        {/* Advanced server settings */}
        <button
          style={advancedToggle}
          type="button"
          onClick={handleToggleAdvanced}
          onMouseEnter={(e) => { e.currentTarget.style.color = "#8888aa"; }}
          onMouseLeave={(e) => { e.currentTarget.style.color = "#555577"; }}
        >
          {showAdvanced ? "بستن تنظیمات پیشرفته" : "تنظیمات پیشرفته"}
        </button>

        {showAdvanced && (
          <div style={advancedSection}>
            <div style={{ fontSize: "0.82rem", color: "#8888aa", marginBottom: "10px", fontWeight: 500 }}>
              آدرس سرور
            </div>
            <div style={serverRow}>
              <input
                style={{ ...inputStyle, marginBottom: 0, flex: 1, direction: "ltr", textAlign: "left" }}
                placeholder="http://server:8080"
                value={serverURL}
                onChange={(e) => setServerURL(e.target.value)}
                onFocus={inputFocusHandler}
                onBlur={inputBlurHandler}
              />
              <button
                type="button"
                style={{
                  padding: "10px 18px",
                  borderRadius: "10px",
                  border: "none",
                  background: "#7c4dff",
                  color: "#fff",
                  cursor: "pointer",
                  fontWeight: 600,
                  fontSize: "0.85rem",
                  flexShrink: 0,
                  fontFamily: "'Vazirmatn', sans-serif",
                }}
                onClick={handleSaveServer}
              >
                ذخیره
              </button>
            </div>
          </div>
        )}
      </form>
    </div>
  );
}
