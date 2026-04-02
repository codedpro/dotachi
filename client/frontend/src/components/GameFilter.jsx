import React from "react";

const GAMES = [
  { key: "all", label: "همه" },
  { key: "dota2", label: "دوتا ۲" },
  { key: "cs2", label: "CS2" },
  { key: "wc3", label: "وارکرافت ۳" },
  { key: "aoe", label: "عصر امپراطوری" },
  { key: "valorant", label: "ولورنت" },
  { key: "mc", label: "ماین‌کرفت" },
  { key: "other", label: "سایر" },
];

const containerStyle = {
  display: "flex",
  gap: "8px",
  flexWrap: "wrap",
  alignItems: "center",
  direction: "rtl",
};

const pillStyle = (active) => ({
  padding: "6px 16px",
  borderRadius: "20px",
  border: active ? "1px solid #7c4dff" : "1px solid #2a2a45",
  background: active ? "linear-gradient(135deg, #7c4dff, #6a3de8)" : "#141428",
  color: active ? "#fff" : "#8888aa",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: active ? 600 : 500,
  transition: "all 0.2s ease",
  outline: "none",
  letterSpacing: "0.3px",
  fontFamily: "'Vazirmatn', sans-serif",
});

export default function GameFilter({ active, onChange }) {
  return (
    <div style={containerStyle}>
      {GAMES.map((g) => (
        <button
          key={g.key}
          style={pillStyle(active === g.key)}
          onClick={() => onChange(g.key)}
          onMouseEnter={(e) => {
            if (active !== g.key) {
              e.currentTarget.style.borderColor = "#7c4dff66";
              e.currentTarget.style.color = "#c0c0d0";
            }
          }}
          onMouseLeave={(e) => {
            if (active !== g.key) {
              e.currentTarget.style.borderColor = "#2a2a45";
              e.currentTarget.style.color = "#8888aa";
            }
          }}
        >
          {g.label}
        </button>
      ))}
    </div>
  );
}

export { GAMES };
