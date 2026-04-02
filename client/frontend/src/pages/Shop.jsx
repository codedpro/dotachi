import React, { useState } from "react";

const SHARD_COLOR = "#ff9800";

const pageStyle = {
  maxWidth: "700px",
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

const heroStyle = {
  textAlign: "center",
  padding: "32px 24px",
};

const heroIcon = {
  fontSize: "3rem",
  marginBottom: "12px",
};

const heroTitle = {
  fontSize: "1.5rem",
  fontWeight: 700,
  color: "#e8e8f0",
  marginBottom: "8px",
};

const heroSub = {
  fontSize: "0.95rem",
  color: "#8888aa",
  lineHeight: 1.6,
};

const tableWrapper = {
  overflowX: "auto",
};

const tableStyle = {
  width: "100%",
  borderCollapse: "separate",
  borderSpacing: "0",
  fontSize: "0.9rem",
};

const thStyle = {
  padding: "12px 16px",
  textAlign: "right",
  color: "#8888aa",
  fontWeight: 600,
  fontSize: "0.8rem",
  letterSpacing: "0.5px",
  borderBottom: "1px solid #2a2a45",
};

const tdStyle = {
  padding: "12px 16px",
  color: "#e8e8f0",
  fontFamily: "monospace",
  borderBottom: "1px solid #1a1a30",
  direction: "ltr",
  textAlign: "center",
};

const discountBadge = (color) => ({
  display: "inline-block",
  padding: "2px 8px",
  borderRadius: "8px",
  fontSize: "0.7rem",
  fontWeight: 600,
  background: `${color}18`,
  color: color,
  marginRight: "6px",
});

const formulaCard = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px 20px",
  border: "1px solid #1a1a30",
  textAlign: "center",
};

const formulaText = {
  fontSize: "1rem",
  color: SHARD_COLOR,
  fontWeight: 600,
  marginBottom: "6px",
};

const formulaSubtext = {
  fontSize: "0.85rem",
  color: "#8888aa",
};

const contactGrid = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "12px",
};

const contactCard = {
  background: "#0d0d20",
  borderRadius: "12px",
  padding: "20px",
  border: "1px solid #2a2a45",
  textAlign: "center",
  cursor: "pointer",
  transition: "all 0.2s ease",
};

const contactIcon = {
  fontSize: "1.5rem",
  marginBottom: "8px",
};

const contactPlatform = {
  fontSize: "0.9rem",
  fontWeight: 600,
  color: "#e8e8f0",
  marginBottom: "4px",
};

const contactHandle = {
  fontSize: "0.85rem",
  color: SHARD_COLOR,
  fontFamily: "monospace",
  direction: "ltr",
};

const quickBuyGrid = {
  display: "grid",
  gridTemplateColumns: "repeat(4, 1fr)",
  gap: "10px",
};

const quickBuyCard = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px 12px",
  border: "1px solid #2a2a45",
  textAlign: "center",
  cursor: "pointer",
  transition: "all 0.2s ease",
};

const quickBuyShards = {
  fontSize: "1.1rem",
  fontWeight: 700,
  color: SHARD_COLOR,
  marginBottom: "4px",
  direction: "ltr",
};

const quickBuyPrice = {
  fontSize: "0.8rem",
  color: "#8888aa",
};

const sharedLanRow = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "14px 20px",
  background: "#0d0d20",
  borderRadius: "10px",
  border: "1px solid #1a1a30",
  marginTop: "12px",
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

function formatNumber(n) {
  return n.toLocaleString("en-US");
}

// Pricing calculations
function calcWeekly(slots) {
  return slots * 1000 * 7;
}
function calcMonthly(slots) {
  return Math.round(slots * 1000 * 30 * 0.9);
}
function calcQuarterly(slots) {
  return Math.round(slots * 1000 * 90 * 0.75);
}
function calcYearly(slots) {
  return Math.round(slots * 1000 * 365 * 0.6);
}

const SLOT_EXAMPLES = [15, 25, 50, 100];
const QUICK_BUY = [
  { shards: 10000, label: "10K" },
  { shards: 50000, label: "50K" },
  { shards: 100000, label: "100K" },
  { shards: 500000, label: "500K" },
];

export default function ShopPage() {
  return (
    <div style={pageStyle}>
      {/* Hero */}
      <div style={{ ...card, ...heroStyle }}>
        <div style={heroIcon}>{"🔶"}</div>
        <div style={heroTitle}>فروشگاه شارد دوتاچی</div>
        <div style={heroSub}>
          شارد بخرید و اتاق بازی بسازید
          <br />
          {"هر شارد = ۱ تومان"}
        </div>
      </div>

      {/* Pricing formula */}
      <div style={card}>
        <div style={sectionTitle}>فرمول قیمت‌گذاری</div>
        <div style={formulaCard}>
          <div style={formulaText}>{"هر اسلات ۱,۰۰۰ شارد در روز"}</div>
          <div style={formulaSubtext}>هر اسلات = 1,000 شارد در روز</div>
        </div>
        <div style={sharedLanRow}>
          <div>
            <div style={{ color: "#e8e8f0", fontWeight: 600, fontSize: "0.95rem" }}>
              {"🔶"} لن اشتراکی (اتاق‌های عمومی)
            </div>
            <div style={{ color: "#8888aa", fontSize: "0.82rem", marginTop: "4px" }}>
              {"ورود به لان عمومی: ۲,۰۰۰ شارد در ساعت"}
            </div>
          </div>
          <div style={{ color: SHARD_COLOR, fontWeight: 700, fontSize: "1.1rem", fontFamily: "monospace", direction: "ltr" }}>
            2,000/ساعت
          </div>
        </div>
      </div>

      {/* Pricing table */}
      <div style={card}>
        <div style={sectionTitle}>نمونه قیمت‌ها</div>
        <div style={tableWrapper}>
          <table style={tableStyle}>
            <thead>
              <tr>
                <th style={thStyle}>ظرفیت</th>
                <th style={thStyle}>۷ روز (حداقل)</th>
                <th style={thStyle}>
                  ماهانه
                  <span style={discountBadge("#00e676")}>۱۰٪-</span>
                </th>
                <th style={thStyle}>
                  ۳ ماهه
                  <span style={discountBadge("#42a5f5")}>۲۵٪-</span>
                </th>
                <th style={thStyle}>
                  سالانه
                  <span style={discountBadge(SHARD_COLOR)}>۴۰٪-</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {SLOT_EXAMPLES.map((slots) => (
                <tr key={slots}>
                  <td style={{ ...tdStyle, fontWeight: 600, color: SHARD_COLOR }}>{slots}</td>
                  <td style={tdStyle}>{"🔶"} {formatNumber(calcWeekly(slots))}</td>
                  <td style={tdStyle}>{"🔶"} {formatNumber(calcMonthly(slots))}</td>
                  <td style={tdStyle}>{"🔶"} {formatNumber(calcQuarterly(slots))}</td>
                  <td style={tdStyle}>{"🔶"} {formatNumber(calcYearly(slots))}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Quick Buy section */}
      <div style={card}>
        <div style={sectionTitle}>خرید سریع</div>
        <div style={quickBuyGrid}>
          {QUICK_BUY.map((item) => (
            <div
              key={item.shards}
              style={quickBuyCard}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = SHARD_COLOR;
                e.currentTarget.style.background = `${SHARD_COLOR}08`;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = "#2a2a45";
                e.currentTarget.style.background = "#0d0d20";
              }}
            >
              <div style={quickBuyShards}>{"🔶"} {item.label}</div>
              <div style={quickBuyPrice}>{formatNumber(item.shards)} تومان</div>
            </div>
          ))}
        </div>
        <div style={{ textAlign: "center", marginTop: "12px", fontSize: "0.82rem", color: "#8888aa" }}>
          برای خرید شارد با ما تماس بگیرید
        </div>
      </div>

      {/* Contact section */}
      <div style={card}>
        <div style={sectionTitle}>خرید شارد - تماس با ما</div>
        <div style={contactGrid}>
          <div
            style={contactCard}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = "#0088cc";
              e.currentTarget.style.background = "#0088cc08";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = "#2a2a45";
              e.currentTarget.style.background = "#0d0d20";
            }}
          >
            <div style={contactIcon}>{"📱"}</div>
            <div style={contactPlatform}>تلگرام</div>
            <div style={contactHandle}>@coded_pro</div>
          </div>
          <div
            style={contactCard}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = "#00b862";
              e.currentTarget.style.background = "#00b86208";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = "#2a2a45";
              e.currentTarget.style.background = "#0d0d20";
            }}
          >
            <div style={contactIcon}>{"💬"}</div>
            <div style={contactPlatform}>بله</div>
            <div style={contactHandle}>@coded_pro</div>
          </div>
        </div>
      </div>
    </div>
  );
}
