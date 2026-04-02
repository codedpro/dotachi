import React, { useState } from "react";

const SHARD_COLOR = "#ff9800";

const GAME_BADGES = {
  dota2: { abbr: "D2", color: "#e74c3c", bg: "#e74c3c18" },
  cs2: { abbr: "CS", color: "#f39c12", bg: "#f39c1218" },
  wc3: { abbr: "WC3", color: "#2ecc71", bg: "#2ecc7118" },
  aoe: { abbr: "AoE", color: "#3498db", bg: "#3498db18" },
  valorant: { abbr: "VAL", color: "#ff4655", bg: "#ff465518" },
  mc: { abbr: "MC", color: "#7fba00", bg: "#7fba0018" },
  other: { abbr: "?", color: "#8888aa", bg: "#8888aa18" },
};

function detectGame(name) {
  const lower = (name || "").toLowerCase();
  if (lower.includes("dota")) return "dota2";
  if (lower.includes("cs2") || lower.includes("counter")) return "cs2";
  if (lower.includes("wc3") || lower.includes("warcraft")) return "wc3";
  if (lower.includes("aoe") || lower.includes("age of")) return "aoe";
  if (lower.includes("valorant") || lower.includes("val")) return "valorant";
  if (lower.includes("minecraft") || lower.includes("mc")) return "mc";
  return "other";
}

function getPingColor(ms) {
  if (ms == null || ms < 0) return "#8888aa";
  if (ms < 50) return "#00e676";
  if (ms < 100) return "#ffab00";
  return "#ff5252";
}

function getExpiryInfo(expiresAt) {
  if (!expiresAt) return null;
  try {
    const exp = new Date(expiresAt);
    const now = new Date();
    const diff = exp - now;
    if (diff <= 0) return { text: "منقضی شده", color: "#ff5252" };
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    if (days > 3) return { text: `${days} روز مانده`, color: "#8888aa" };
    if (days >= 1) return { text: `${days} روز ${hours} ساعت`, color: SHARD_COLOR };
    return { text: `${hours} ساعت مانده`, color: "#ff5252" };
  } catch {
    return null;
  }
}

export default function RoomCard({ room, onJoin, pingMs, isFavorite, onToggleFavorite }) {
  const [hovered, setHovered] = useState(false);

  const isFull = room.current_players >= room.max_players;
  const game = room.game_tag ? room.game_tag : detectGame(room.name);
  const badge = GAME_BADGES[game] || GAME_BADGES.other;
  const playerPct = room.max_players > 0
    ? (room.current_players / room.max_players) * 100
    : 0;

  const isShared = !room.is_private && !room.owner_id;
  const expiry = getExpiryInfo(room.expires_at);

  const cardStyle = {
    background: "#1a1a35",
    borderRadius: "12px",
    padding: "20px",
    display: "flex",
    flexDirection: "column",
    gap: "12px",
    transition: "transform 0.2s ease, box-shadow 0.2s ease",
    cursor: "default",
    border: "1px solid #2a2a45",
    transform: hovered ? "translateY(-2px)" : "translateY(0)",
    boxShadow: hovered
      ? "0 8px 24px rgba(124, 77, 255, 0.12)"
      : "0 2px 8px rgba(0, 0, 0, 0.2)",
    position: "relative",
    overflow: "hidden",
    direction: "rtl",
    textAlign: "right",
  };

  return (
    <div
      style={cardStyle}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Top row: game badge + name + badges + star */}
      <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
        {/* Game badge */}
        <div
          style={{
            width: "36px",
            height: "36px",
            borderRadius: "8px",
            background: badge.bg,
            border: `1px solid ${badge.color}33`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: "0.7rem",
            fontWeight: 700,
            color: badge.color,
            flexShrink: 0,
            letterSpacing: "0.5px",
          }}
        >
          {badge.abbr}
        </div>

        {/* Room name */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: "1rem",
              fontWeight: 600,
              color: "#e8e8f0",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              display: "flex",
              alignItems: "center",
              gap: "6px",
            }}
          >
            {room.is_private && (
              <span style={{ color: "#ffab00", fontSize: "0.8rem" }} title="اتاق خصوصی">
                &#128274;
              </span>
            )}
            {room.name}
          </div>
          <div style={{ fontSize: "0.8rem", color: "#8888aa", marginTop: "2px", display: "flex", alignItems: "center", gap: "6px" }}>
            {isShared ? (
              <span style={{ color: SHARD_COLOR, fontWeight: 600, fontFamily: "monospace", direction: "ltr" }}>
                {"🔶"} 2,000/ساعت
              </span>
            ) : (
              room.owner_display_name
            )}
          </div>
        </div>

        {/* Badges column */}
        <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-start", gap: "4px", flexShrink: 0 }}>
          {isShared && (
            <span style={{
              padding: "2px 8px",
              borderRadius: "6px",
              fontSize: "0.65rem",
              fontWeight: 700,
              background: "#2196f318",
              color: "#2196f3",
              border: "1px solid #2196f333",
              letterSpacing: "0.5px",
            }}>
              اشتراکی
            </span>
          )}
          {expiry && (
            <span style={{
              padding: "2px 8px",
              borderRadius: "6px",
              fontSize: "0.65rem",
              fontWeight: 600,
              background: `${expiry.color}12`,
              color: expiry.color,
              border: `1px solid ${expiry.color}22`,
              fontFamily: "monospace",
              direction: "ltr",
            }}>
              {expiry.text}
            </span>
          )}
        </div>

        {/* Favorite star */}
        {onToggleFavorite && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleFavorite(room.id);
            }}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              fontSize: "1.1rem",
              color: isFavorite ? "#ffab00" : "#444466",
              padding: "4px",
              transition: "color 0.2s ease",
              flexShrink: 0,
            }}
            title={isFavorite ? "حذف از علاقه‌مندی‌ها" : "افزودن به علاقه‌مندی‌ها"}
          >
            {isFavorite ? "\u2605" : "\u2606"}
          </button>
        )}
      </div>

      {/* Player count bar */}
      <div>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "6px",
          }}
        >
          <span style={{ fontSize: "0.8rem", color: "#8888aa" }}>بازیکنان</span>
          <span
            style={{
              fontSize: "0.85rem",
              fontWeight: 600,
              color: isFull ? "#ff5252" : "#e8e8f0",
              fontFamily: "monospace",
              direction: "ltr",
            }}
          >
            {room.current_players}/{room.max_players}
          </span>
        </div>
        <div
          style={{
            height: "4px",
            background: "#0d0d20",
            borderRadius: "2px",
            overflow: "hidden",
          }}
        >
          <div
            style={{
              height: "100%",
              width: `${playerPct}%`,
              background: isFull
                ? "#ff5252"
                : playerPct > 70
                ? "#ffab00"
                : "#7c4dff",
              borderRadius: "2px",
              transition: "width 0.3s ease",
              float: "left",
            }}
          />
        </div>
      </div>

      {/* Bottom row: ping + join button */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginTop: "auto",
        }}
      >
        {/* Ping indicator */}
        <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
          <div
            style={{
              width: "8px",
              height: "8px",
              borderRadius: "50%",
              background: getPingColor(pingMs),
              boxShadow: `0 0 6px ${getPingColor(pingMs)}66`,
            }}
          />
          <span
            style={{
              fontSize: "0.8rem",
              fontFamily: "monospace",
              color: getPingColor(pingMs),
              fontWeight: 500,
              direction: "ltr",
            }}
          >
            {pingMs != null && pingMs >= 0 ? `${pingMs}ms` : "--"}
          </span>
        </div>

        {/* Join button */}
        <button
          style={{
            padding: "7px 20px",
            borderRadius: "8px",
            border: "none",
            background: isFull
              ? "#2a2a45"
              : hovered
              ? "linear-gradient(135deg, #9c6dff, #7c4dff)"
              : "#7c4dff",
            color: isFull ? "#666" : "#fff",
            fontSize: "0.85rem",
            fontWeight: 600,
            cursor: isFull ? "not-allowed" : "pointer",
            transition: "all 0.2s ease",
            letterSpacing: "0.3px",
            fontFamily: "'Vazirmatn', sans-serif",
          }}
          disabled={isFull}
          onClick={onJoin}
        >
          {isFull ? "پر" : "ورود"}
        </button>
      </div>
    </div>
  );
}

export { detectGame, GAME_BADGES };
