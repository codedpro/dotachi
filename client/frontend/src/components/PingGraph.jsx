import React from "react";

const containerStyle = {
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  direction: "rtl",
};

const headerStyle = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
};

const titleStyle = {
  fontSize: "0.8rem",
  fontWeight: 600,
  color: "#8888aa",
  letterSpacing: "0.5px",
};

const valueStyle = {
  fontSize: "0.8rem",
  color: "#8888aa",
  fontFamily: "monospace",
  direction: "ltr",
};

const graphContainer = {
  display: "flex",
  alignItems: "flex-end",
  gap: "2px",
  height: "60px",
  padding: "4px 0",
  background: "#0d0d20",
  borderRadius: "8px",
  paddingLeft: "4px",
  paddingRight: "4px",
};

function getBarColor(ms) {
  if (ms < 0) return "#ff525266";
  if (ms < 50) return "#00e676";
  if (ms < 100) return "#ffab00";
  return "#ff5252";
}

export default function PingGraph({ pings }) {
  // pings is an array of last N ping values in ms (-1 = failed)
  const data = pings || [];
  const maxPing = Math.max(1, ...data.filter((p) => p > 0));
  const barCount = 20;

  // Pad to barCount entries
  const padded = [];
  for (let i = 0; i < barCount; i++) {
    const idx = data.length - barCount + i;
    padded.push(idx >= 0 ? data[idx] : -2); // -2 = no data
  }

  const latestValid = [...data].reverse().find((p) => p >= 0);

  return (
    <div style={containerStyle}>
      <div style={headerStyle}>
        <span style={titleStyle}>تاریخچه پینگ</span>
        {latestValid !== undefined && (
          <span style={valueStyle}>آخرین: {latestValid}ms</span>
        )}
      </div>
      <div style={graphContainer}>
        {padded.map((ms, i) => {
          if (ms === -2) {
            // No data -- empty slot
            return (
              <div
                key={i}
                style={{
                  flex: 1,
                  height: "2px",
                  background: "#1a1a35",
                  borderRadius: "1px",
                }}
              />
            );
          }
          if (ms < 0) {
            // Failed ping -- red X marker
            return (
              <div
                key={i}
                style={{
                  flex: 1,
                  height: "100%",
                  display: "flex",
                  alignItems: "flex-end",
                  justifyContent: "center",
                }}
              >
                <div
                  style={{
                    width: "100%",
                    height: "4px",
                    background: "#ff525266",
                    borderRadius: "1px",
                  }}
                />
              </div>
            );
          }
          const heightPct = Math.max(8, (ms / maxPing) * 100);
          return (
            <div
              key={i}
              style={{
                flex: 1,
                height: `${heightPct}%`,
                background: getBarColor(ms),
                borderRadius: "2px 2px 0 0",
                minHeight: "4px",
                transition: "height 0.3s ease",
                opacity: i === padded.length - 1 ? 1 : 0.75,
              }}
              title={`${ms}ms`}
            />
          );
        })}
      </div>
    </div>
  );
}
