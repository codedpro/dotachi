import React, { useState, useEffect, useRef } from "react";
import { GetMyStats, ChangePassword, RedeemPromo, GetReferralInfo, RefreshBalance } from "../api";

const SHARD_COLOR = "#ff9800";

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

const profileHeader = {
  display: "flex",
  alignItems: "center",
  gap: "20px",
  marginBottom: "20px",
};

const avatarStyle = {
  width: "64px",
  height: "64px",
  borderRadius: "16px",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: "1.5rem",
  fontWeight: 700,
  color: "#fff",
  flexShrink: 0,
  boxShadow: "0 4px 16px rgba(124, 77, 255, 0.25)",
};

const nameStyle = {
  fontSize: "1.3rem",
  fontWeight: 700,
  color: "#e8e8f0",
};

const shardHeroStyle = {
  textAlign: "center",
  padding: "24px",
  background: `${SHARD_COLOR}08`,
  borderRadius: "12px",
  border: `1px solid ${SHARD_COLOR}22`,
};

const shardBigNumber = {
  fontSize: "2rem",
  fontWeight: 800,
  color: SHARD_COLOR,
  fontFamily: "monospace",
  marginBottom: "4px",
  direction: "ltr",
};

const shardLabel = {
  fontSize: "0.9rem",
  color: "#8888aa",
  fontWeight: 500,
};

const buyBtn = {
  marginTop: "12px",
  padding: "10px 28px",
  borderRadius: "10px",
  border: `1px solid ${SHARD_COLOR}`,
  background: `linear-gradient(135deg, ${SHARD_COLOR}, #e68900)`,
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.9rem",
  boxShadow: `0 4px 12px ${SHARD_COLOR}33`,
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

const statGrid = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr 1fr",
  gap: "12px",
};

const statBox = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px",
  border: "1px solid #1a1a30",
  textAlign: "center",
};

const statLabelStyle = {
  fontSize: "0.72rem",
  color: "#8888aa",
  letterSpacing: "0.5px",
  marginBottom: "8px",
};

const statValueStyle = {
  fontSize: "1.2rem",
  fontWeight: 700,
  color: "#e8e8f0",
  fontFamily: "monospace",
  direction: "ltr",
};

const inputStyle = {
  width: "100%",
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.95rem",
  outline: "none",
  direction: "rtl",
  textAlign: "right",
  fontFamily: "'Vazirmatn', sans-serif",
};

const btnPrimary = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "none",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.88rem",
  boxShadow: "0 4px 12px rgba(124, 77, 255, 0.2)",
  fontFamily: "'Vazirmatn', sans-serif",
};

const copyBtn = {
  padding: "6px 14px",
  borderRadius: "8px",
  border: "1px solid #2a2a45",
  background: "#141428",
  color: "#8888aa",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 500,
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
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

function formatShards(n) {
  if (n == null) return "0";
  return n.toLocaleString("en-US");
}

export default function ProfilePage({ user, shardBalance, onNavigateShop, onShardUpdate }) {
  const [editName, setEditName] = useState(user?.display_name || "");
  const [editing, setEditing] = useState(false);
  const [stats, setStats] = useState(null);
  const [offline, setOffline] = useState(false);

  // Password change state
  const [oldPass, setOldPass] = useState("");
  const [newPass, setNewPass] = useState("");
  const [passMsg, setPassMsg] = useState("");
  const [passMsgColor, setPassMsgColor] = useState("#00e676");
  const [changingPass, setChangingPass] = useState(false);

  // Promo code state
  const [promoCode, setPromoCode] = useState("");
  const [promoMsg, setPromoMsg] = useState("");
  const [promoMsgColor, setPromoMsgColor] = useState("#00e676");
  const [redeemingPromo, setRedeemingPromo] = useState(false);

  // Referral state
  const [referralInfo, setReferralInfo] = useState(null);
  const [referralCopied, setReferralCopied] = useState(false);
  const referralCopyTimerRef = useRef(null);

  // Refresh balance state
  const [refreshingBalance, setRefreshingBalance] = useState(false);

  // Derive initials
  const initials = (user?.display_name || "?")
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  // Load stats and referral info
  useEffect(() => {
    (async () => {
      try {
        const s = await GetMyStats();
        if (s) setStats(s);
        setOffline(false);
      } catch (err) {
        const errStr = String(err);
        if (errStr.includes("fetch") || errStr.includes("network") || errStr.includes("Failed") || errStr.includes("ECONNREFUSED")) {
          setOffline(true);
        }
      }
      try {
        const ref = await GetReferralInfo();
        if (ref) setReferralInfo(ref);
      } catch {
        // ignore
      }
    })();
    return () => {
      if (referralCopyTimerRef.current) clearTimeout(referralCopyTimerRef.current);
    };
  }, []);

  const handleChangePassword = async () => {
    if (!oldPass || !newPass) return;
    setChangingPass(true);
    setPassMsg("");
    try {
      await ChangePassword(oldPass, newPass);
      setPassMsg("\u0631\u0645\u0632 \u0639\u0628\u0648\u0631 \u0628\u0627 \u0645\u0648\u0641\u0642\u06CC\u062A \u062A\u063A\u06CC\u06CC\u0631 \u06A9\u0631\u062F");
      setPassMsgColor("#00e676");
      setOldPass("");
      setNewPass("");
    } catch (err) {
      setPassMsg(String(err) || "\u062E\u0637\u0627 \u062F\u0631 \u062A\u063A\u06CC\u06CC\u0631 \u0631\u0645\u0632 \u0639\u0628\u0648\u0631");
      setPassMsgColor("#ff5252");
    } finally {
      setChangingPass(false);
    }
  };

  const handleRedeemPromo = async () => {
    if (!promoCode.trim()) return;
    setRedeemingPromo(true);
    setPromoMsg("");
    try {
      const result = await RedeemPromo(promoCode.trim());
      setPromoMsg("\u06A9\u062F \u062A\u062E\u0641\u06CC\u0641 \u0628\u0627 \u0645\u0648\u0641\u0642\u06CC\u062A \u0627\u0639\u0645\u0627\u0644 \u0634\u062F");
      setPromoMsgColor("#00e676");
      setPromoCode("");
      if (result && result.new_balance !== undefined && onShardUpdate) {
        onShardUpdate(result.new_balance);
      }
    } catch (err) {
      setPromoMsg(String(err) || "\u06A9\u062F \u0646\u0627\u0645\u0639\u062A\u0628\u0631 \u0627\u0633\u062A");
      setPromoMsgColor("#ff5252");
    } finally {
      setRedeemingPromo(false);
    }
  };

  const handleCopyReferral = () => {
    if (!referralInfo?.code) return;
    try {
      navigator.clipboard.writeText(referralInfo.code);
      setReferralCopied(true);
      if (referralCopyTimerRef.current) clearTimeout(referralCopyTimerRef.current);
      referralCopyTimerRef.current = setTimeout(() => setReferralCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  const handleRefreshBalance = async () => {
    setRefreshingBalance(true);
    try {
      const result = await RefreshBalance();
      if (result && result.balance !== undefined && onShardUpdate) {
        onShardUpdate(result.balance);
      }
    } catch {
      // ignore
    } finally {
      setRefreshingBalance(false);
    }
  };

  return (
    <div style={pageStyle}>
      {offline && (
        <div style={offlineBanner}>
          سرور در دسترس نیست. تلگرام: @coded_pro
        </div>
      )}

      {/* Profile header card */}
      <div style={card}>
        <div style={profileHeader}>
          <div style={avatarStyle}>{initials}</div>
          <div>
            <div style={nameStyle}>{user?.display_name || "ناشناس"}</div>
            <div style={{ display: "flex", alignItems: "center", gap: "8px", marginTop: "6px" }}>
              {user?.phone && (
                <span style={{ color: "#555577", fontSize: "0.82rem", fontFamily: "monospace", direction: "ltr" }}>
                  {user.phone}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Shard Balance */}
      <div style={card}>
        <div style={shardHeroStyle}>
          <div style={{ fontSize: "2rem", marginBottom: "8px" }}>{"🔶"}</div>
          <div style={shardBigNumber}>{formatShards(shardBalance)}</div>
          <div style={shardLabel}>موجودی شارد</div>
          <button
            style={buyBtn}
            onClick={onNavigateShop}
            onMouseEnter={(e) => {
              e.currentTarget.style.boxShadow = `0 6px 20px ${SHARD_COLOR}44`;
              e.currentTarget.style.transform = "translateY(-1px)";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.boxShadow = `0 4px 12px ${SHARD_COLOR}33`;
              e.currentTarget.style.transform = "translateY(0)";
            }}
          >
            خرید شارد
          </button>
        </div>
      </div>

      {/* Stats */}
      <div style={card}>
        <div style={sectionTitle}>آمار</div>
        <div style={statGrid}>
          <div style={statBox}>
            <div style={statLabelStyle}>تعداد بازی</div>
            <div style={statValueStyle}>{stats?.sessions ?? "--"}</div>
          </div>
          <div style={statBox}>
            <div style={statLabelStyle}>ساعت بازی</div>
            <div style={statValueStyle}>{stats?.play_hours ?? "--"}</div>
          </div>
          <div style={statBox}>
            <div style={statLabelStyle}>اتاق‌های شما</div>
            <div style={statValueStyle}>{stats?.rooms_owned ?? "--"}</div>
          </div>
        </div>
      </div>

      {/* Edit display name */}
      <div style={card}>
        <div style={sectionTitle}>ویرایش پروفایل</div>
        <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
          <input
            style={{ ...inputStyle, flex: 1 }}
            type="text"
            placeholder="نام نمایشی"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            disabled={!editing}
          />
          {!editing ? (
            <button style={copyBtn} onClick={() => setEditing(true)}>
              ویرایش
            </button>
          ) : (
            <button
              style={btnPrimary}
              onClick={() => {
                // In the future, call an API to update display name
                setEditing(false);
              }}
            >
              ذخیره
            </button>
          )}
        </div>
      </div>

      {/* Password Change */}
      <div style={card}>
        <div style={sectionTitle}>{"\u062A\u063A\u06CC\u06CC\u0631 \u0631\u0645\u0632 \u0639\u0628\u0648\u0631"}</div>
        <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
          <input
            style={inputStyle}
            type="password"
            placeholder={"\u0631\u0645\u0632 \u0639\u0628\u0648\u0631 \u0641\u0639\u0644\u06CC"}
            value={oldPass}
            onChange={(e) => setOldPass(e.target.value)}
          />
          <input
            style={inputStyle}
            type="password"
            placeholder={"\u0631\u0645\u0632 \u0639\u0628\u0648\u0631 \u062C\u062F\u06CC\u062F"}
            value={newPass}
            onChange={(e) => setNewPass(e.target.value)}
          />
          <button
            style={{ ...btnPrimary, opacity: changingPass ? 0.7 : 1 }}
            onClick={handleChangePassword}
            disabled={changingPass || !oldPass || !newPass}
          >
            {changingPass ? "\u062F\u0631 \u062D\u0627\u0644 \u062A\u063A\u06CC\u06CC\u0631..." : "\u0630\u062E\u06CC\u0631\u0647"}
          </button>
          {passMsg && (
            <div style={{ fontSize: "0.82rem", color: passMsgColor, marginTop: "4px" }}>
              {passMsg}
            </div>
          )}
        </div>
      </div>

      {/* Promo Code Redemption */}
      <div style={card}>
        <div style={sectionTitle}>{"\u06A9\u062F \u062A\u062E\u0641\u06CC\u0641"}</div>
        <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
          <input
            style={{ ...inputStyle, flex: 1, direction: "ltr", textAlign: "left" }}
            type="text"
            placeholder={"\u06A9\u062F \u062A\u062E\u0641\u06CC\u0641 \u0631\u0627 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F"}
            value={promoCode}
            onChange={(e) => setPromoCode(e.target.value)}
          />
          <button
            style={{ ...btnPrimary, opacity: redeemingPromo ? 0.7 : 1 }}
            onClick={handleRedeemPromo}
            disabled={redeemingPromo || !promoCode.trim()}
          >
            {redeemingPromo ? "\u062F\u0631 \u062D\u0627\u0644 \u0627\u0639\u0645\u0627\u0644..." : "\u0627\u0633\u062A\u0641\u0627\u062F\u0647"}
          </button>
        </div>
        {promoMsg && (
          <div style={{ fontSize: "0.82rem", color: promoMsgColor, marginTop: "8px" }}>
            {promoMsg}
          </div>
        )}
      </div>

      {/* Referral Code */}
      <div style={card}>
        <div style={sectionTitle}>{"\u06A9\u062F \u0645\u0639\u0631\u0641\u06CC"}</div>
        {referralInfo?.code ? (
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
              <div
                onClick={handleCopyReferral}
                style={{
                  flex: 1,
                  background: "#0d0d20",
                  borderRadius: "8px",
                  padding: "12px 14px",
                  border: "1px solid #2a2a45",
                  fontFamily: "monospace",
                  fontSize: "1rem",
                  color: referralCopied ? "#00e676" : "#e8e8f0",
                  cursor: "pointer",
                  transition: "color 0.3s",
                  userSelect: "all",
                  direction: "ltr",
                  textAlign: "center",
                  fontWeight: 700,
                }}
              >
                {referralInfo.code}
              </div>
              <button style={copyBtn} onClick={handleCopyReferral}>
                {referralCopied ? "\u06A9\u067E\u06CC \u0634\u062F" : "\u06A9\u067E\u06CC"}
              </button>
            </div>
            {referralInfo.total_referrals !== undefined && (
              <div style={{ fontSize: "0.82rem", color: "#8888aa", marginTop: "10px" }}>
                {"\u062A\u0639\u062F\u0627\u062F \u0645\u0639\u0631\u0641\u06CC\u200C\u0647\u0627: "}
                <span style={{ color: "#e8e8f0", fontWeight: 600, fontFamily: "monospace" }}>
                  {referralInfo.total_referrals}
                </span>
              </div>
            )}
            {referralInfo.earned_shards !== undefined && (
              <div style={{ fontSize: "0.82rem", color: "#8888aa", marginTop: "4px" }}>
                {"\u0634\u0627\u0631\u062F \u06A9\u0633\u0628 \u0634\u062F\u0647: "}
                <span style={{ color: SHARD_COLOR, fontWeight: 600, fontFamily: "monospace" }}>
                  {formatShards(referralInfo.earned_shards)}
                </span>
              </div>
            )}
          </div>
        ) : (
          <div style={{ color: "#555577", fontSize: "0.85rem" }}>
            {"\u062F\u0631 \u062D\u0627\u0644 \u0628\u0627\u0631\u06AF\u0630\u0627\u0631\u06CC..."}
          </div>
        )}
      </div>

      {/* Refresh Balance */}
      <div style={card}>
        <div style={sectionTitle}>{"\u0645\u0648\u062C\u0648\u062F\u06CC"}</div>
        <button
          style={{ ...btnPrimary, width: "100%", opacity: refreshingBalance ? 0.7 : 1 }}
          onClick={handleRefreshBalance}
          disabled={refreshingBalance}
        >
          {refreshingBalance ? "\u062F\u0631 \u062D\u0627\u0644 \u0628\u0631\u0648\u0632\u0631\u0633\u0627\u0646\u06CC..." : "\u0628\u0631\u0648\u0632\u0631\u0633\u0627\u0646\u06CC \u0645\u0648\u062C\u0648\u062F\u06CC"}
        </button>
      </div>
    </div>
  );
}
