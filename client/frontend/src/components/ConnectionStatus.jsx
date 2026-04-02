import React from "react";

const STATUS_CONFIG = {
  connected: { color: "#00e676", label: "متصل", animate: false },
  connecting: { color: "#ffab00", label: "در حال اتصال...", animate: true },
  reconnecting: { color: "#ff9800", label: "اتصال مجدد...", animate: true },
  disconnected: { color: "#ff5252", label: "قطع شده", animate: false },
};

const wrapperStyle = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
  padding: "16px 20px",
  background: "#0d0d20",
  borderRadius: "10px",
  marginBottom: "16px",
  direction: "rtl",
  textAlign: "right",
};

const dotOuter = (color, animate) => ({
  width: "16px",
  height: "16px",
  borderRadius: "50%",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  background: `${color}22`,
  flexShrink: 0,
  animation: animate ? "pulse 1.5s ease-in-out infinite" : "none",
});

const dotInner = (color) => ({
  width: "8px",
  height: "8px",
  borderRadius: "50%",
  background: color,
  boxShadow: `0 0 10px ${color}88`,
});

const labelStyle = {
  fontSize: "1rem",
  fontWeight: 600,
  color: "#e8e8f0",
};

const pingStyle = (color) => ({
  marginRight: "auto",
  marginLeft: "0",
  fontSize: "0.9rem",
  fontWeight: 600,
  fontFamily: "monospace",
  color: color,
  direction: "ltr",
});

export default function ConnectionStatus({ status, pingMs }) {
  const cfg = STATUS_CONFIG[status] || STATUS_CONFIG.disconnected;

  return (
    <div style={wrapperStyle}>
      <div style={dotOuter(cfg.color, cfg.animate)}>
        <div style={dotInner(cfg.color)} />
      </div>
      <span style={labelStyle}>{cfg.label}</span>
      {status === "connected" && pingMs != null && pingMs >= 0 && (
        <span style={pingStyle(pingMs < 50 ? "#00e676" : pingMs < 100 ? "#ffab00" : "#ff5252")}>
          {pingMs}ms
        </span>
      )}
    </div>
  );
}
