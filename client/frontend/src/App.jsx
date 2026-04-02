import React, { useState, useCallback, useEffect } from "react";
import { StopVPN, GetVPNStatus, GetPingStats, GetConnectionQuality, GetMe, JoinByInvite } from "./api";
import Login from "./pages/Login";
import Rooms from "./pages/Rooms";
import Room from "./pages/Room";
import Profile from "./pages/Profile";
import Settings from "./pages/Settings";
import Shop from "./pages/Shop";
import GameGuides from "./pages/GameGuides";

/*
  Pages: "login" | "rooms" | "favorites" | "room" | "profile" | "shop" | "settings" | "guides"
  Layout: sidebar (when logged in) + main content area
  Direction: RTL
*/

// --- Color tokens ---
const COLORS = {
  bg: "#0a0a1a",
  surface: "#141428",
  card: "#1a1a35",
  accent: "#7c4dff",
  accentHover: "#9c6dff",
  success: "#00e676",
  warning: "#ffab00",
  error: "#ff5252",
  shard: "#ff9800",
  text: "#e8e8f0",
  textSecondary: "#8888aa",
  border: "#2a2a45",
};

// --- Layout styles ---
const appStyle = {
  height: "100vh",
  display: "flex",
  flexDirection: "column",
  background: COLORS.bg,
  overflow: "hidden",
  direction: "rtl",
  fontFamily: "'Vazirmatn', sans-serif",
};

const bodyStyle = {
  flex: 1,
  display: "flex",
  overflow: "hidden",
};

// Sidebar -- on the RIGHT side (RTL flow puts it right automatically)
const sidebarStyle = {
  width: "64px",
  background: COLORS.surface,
  borderLeft: `1px solid ${COLORS.border}`,
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  paddingTop: "16px",
  paddingBottom: "16px",
  gap: "4px",
  flexShrink: 0,
};

const sidebarLogoStyle = {
  width: "36px",
  height: "36px",
  borderRadius: "10px",
  background: `linear-gradient(135deg, ${COLORS.accent}, #6a3de8)`,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  color: "#fff",
  fontSize: "0.7rem",
  fontWeight: 800,
  letterSpacing: "0.5px",
  marginBottom: "20px",
  boxShadow: `0 4px 12px ${COLORS.accent}33`,
  cursor: "pointer",
};

const navBtnStyle = (active) => ({
  width: "44px",
  height: "44px",
  borderRadius: "12px",
  border: "none",
  background: active ? `${COLORS.accent}22` : "transparent",
  color: active ? COLORS.accent : COLORS.textSecondary,
  cursor: "pointer",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontSize: "1.1rem",
  transition: "all 0.2s ease",
  position: "relative",
});

const navTooltip = {
  position: "absolute",
  left: "52px",
  background: COLORS.card,
  color: COLORS.text,
  padding: "4px 10px",
  borderRadius: "6px",
  fontSize: "0.75rem",
  fontWeight: 500,
  whiteSpace: "nowrap",
  pointerEvents: "none",
  border: `1px solid ${COLORS.border}`,
  boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
  zIndex: 100,
  fontFamily: "'Vazirmatn', sans-serif",
};

const sidebarSpacer = { flex: 1 };

// Header
const headerStyle = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "0 24px",
  height: "52px",
  background: COLORS.surface,
  borderBottom: `1px solid ${COLORS.border}`,
  flexShrink: 0,
  direction: "rtl",
};

const headerRight = {
  display: "flex",
  alignItems: "center",
  gap: "16px",
};

const headerTitle = {
  fontSize: "1rem",
  fontWeight: 600,
  color: COLORS.text,
  letterSpacing: "0.5px",
};

const headerLeft = {
  display: "flex",
  alignItems: "center",
  gap: "16px",
};

const shardDisplayStyle = {
  display: "flex",
  alignItems: "center",
  gap: "6px",
  padding: "4px 12px",
  borderRadius: "20px",
  background: `${COLORS.shard}12`,
  border: `1px solid ${COLORS.shard}33`,
  cursor: "pointer",
  transition: "all 0.2s ease",
};

const shardAmountStyle = {
  fontSize: "0.88rem",
  fontWeight: 600,
  color: COLORS.shard,
  fontFamily: "monospace",
  direction: "ltr",
};

const pingDisplay = (ms) => {
  let color = COLORS.textSecondary;
  if (ms != null && ms >= 0) {
    color = ms < 50 ? COLORS.success : ms < 100 ? COLORS.warning : COLORS.error;
  }
  return {
    display: "flex",
    alignItems: "center",
    gap: "6px",
    fontSize: "0.82rem",
    fontFamily: "monospace",
    color: color,
    fontWeight: 500,
    direction: "ltr",
  };
};

const pingDot = (ms) => {
  let color = COLORS.textSecondary;
  if (ms != null && ms >= 0) {
    color = ms < 50 ? COLORS.success : ms < 100 ? COLORS.warning : COLORS.error;
  }
  return {
    width: "6px",
    height: "6px",
    borderRadius: "50%",
    background: color,
    boxShadow: `0 0 4px ${color}66`,
  };
};

const userInfoStyle = {
  fontSize: "0.88rem",
  color: COLORS.text,
  fontWeight: 500,
};

const btnFlat = {
  background: "none",
  border: "none",
  color: COLORS.textSecondary,
  cursor: "pointer",
  fontSize: "0.82rem",
  padding: "6px 12px",
  borderRadius: "8px",
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

// Main content
const mainStyle = {
  flex: 1,
  padding: "20px 24px",
  overflowY: "auto",
  overflowX: "hidden",
};

// Footer
const footerStyle = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "0 24px",
  height: "32px",
  background: COLORS.surface,
  borderTop: `1px solid ${COLORS.border}`,
  flexShrink: 0,
  fontSize: "0.72rem",
  color: "#555577",
  direction: "rtl",
};

const QUALITY_LABELS = {
  excellent: "عالی",
  good: "خوب",
  fair: "متوسط",
  poor: "ضعیف",
};

const qualityIndicator = (quality) => {
  const colors = {
    excellent: COLORS.success,
    good: COLORS.accent,
    fair: COLORS.warning,
    poor: COLORS.error,
  };
  const color = colors[quality] || COLORS.textSecondary;
  return {
    display: "flex",
    alignItems: "center",
    gap: "6px",
    color: color,
  };
};

const qualityDots = (quality) => {
  const levels = { excellent: 4, good: 3, fair: 2, poor: 1 };
  const level = levels[quality] || 0;
  const colors = {
    excellent: COLORS.success,
    good: COLORS.accent,
    fair: COLORS.warning,
    poor: COLORS.error,
  };
  const color = colors[quality] || "#333355";
  return { level, color };
};

function formatShards(n) {
  if (n == null) return "0";
  return n.toLocaleString("en-US");
}

// Navigation items
const NAV_ITEMS = [
  { key: "rooms", label: "اتاق‌ها", icon: "\u25A6" },
  { key: "favorites", label: "علاقه‌مندی‌ها", icon: "\u2605" },
  { key: "guides", label: "راهنما", icon: "\uD83D\uDCD6" },
  { key: "shop", label: "فروشگاه", icon: "\u25C7" },
  { key: "profile", label: "پروفایل", icon: "\u263A" },
  { key: "settings", label: "تنظیمات", icon: "\u2699" },
];

const PAGE_TITLES = {
  rooms: "اتاق‌ها",
  favorites: "علاقه‌مندی‌ها",
  room: "اتاق",
  profile: "پروفایل",
  shop: "فروشگاه شارد",
  settings: "تنظیمات",
  guides: "راهنمای بازی‌ها",
};

export default function App() {
  const [page, setPage] = useState("login");
  const [user, setUser] = useState(null);
  const [shardBalance, setShardBalance] = useState(0);
  const [activeRoom, setActiveRoom] = useState(null);
  const [vpnCreds, setVpnCreds] = useState(null);
  const [hoveredNav, setHoveredNav] = useState(null);
  const [pingMs, setPingMs] = useState(null);
  const [connQuality, setConnQuality] = useState(null);
  const [favorites, setFavorites] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem("dotachi_favorites") || "[]");
    } catch {
      return [];
    }
  });

  // Check for dotachi://join/TOKEN deep link on startup
  useEffect(() => {
    (async () => {
      try {
        // Wails passes CLI args via window. Check for invite token.
        const args = window?.wails?.args || [];
        for (const arg of args) {
          const match = arg.match(/dotachi:\/\/join\/(.+)/);
          if (match && match[1]) {
            const result = await JoinByInvite(match[1]);
            if (result && result.room) {
              setActiveRoom(result.room);
              if (result.vpn_creds) setVpnCreds(result.vpn_creds);
              setPage("room");
            }
            break;
          }
        }
      } catch {
        // ignore -- no invite or not logged in yet
      }
    })();
  }, [user]); // re-check when user logs in

  // Poll ping & quality when in a room
  useEffect(() => {
    if (page !== "room") {
      setPingMs(null);
      setConnQuality(null);
      return;
    }
    const poll = setInterval(async () => {
      try {
        const ps = await GetPingStats();
        if (ps) setPingMs(ps.last_ping);
      } catch {
        // ignore
      }
      try {
        const q = await GetConnectionQuality();
        if (q) setConnQuality(q);
      } catch {
        // ignore
      }
    }, 3000);
    return () => clearInterval(poll);
  }, [page]);

  // Save favorites
  useEffect(() => {
    try {
      localStorage.setItem("dotachi_favorites", JSON.stringify(favorites));
    } catch {
      // ignore
    }
  }, [favorites]);

  // Refresh shard balance periodically when logged in
  const refreshShards = useCallback(async () => {
    try {
      const me = await GetMe();
      if (me && me.shard_balance !== undefined) {
        setShardBalance(me.shard_balance);
      }
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    if (!user) return;
    // Refresh every 30 seconds
    const interval = setInterval(refreshShards, 30000);
    return () => clearInterval(interval);
  }, [user, refreshShards]);

  const toggleFavorite = useCallback((roomId) => {
    setFavorites((prev) =>
      prev.includes(roomId)
        ? prev.filter((id) => id !== roomId)
        : [...prev, roomId]
    );
  }, []);

  const handleLogin = useCallback((tokenResp) => {
    setUser({
      user_id: tokenResp.user_id,
      display_name: tokenResp.display_name,
      is_admin: tokenResp.is_admin,
    });
    setShardBalance(tokenResp.shard_balance || 0);
    setPage("rooms");
  }, []);

  const handleJoin = useCallback((room, creds) => {
    setActiveRoom(room);
    setVpnCreds(creds);
    setPage("room");
  }, []);

  const handleLeave = useCallback(async () => {
    try {
      await StopVPN();
    } catch {
      // best-effort
    }
    setActiveRoom(null);
    setVpnCreds(null);
    setPage("rooms");
  }, []);

  const handleLogout = useCallback(async () => {
    try {
      await StopVPN();
    } catch {
      // best-effort
    }
    setUser(null);
    setShardBalance(0);
    setActiveRoom(null);
    setVpnCreds(null);
    setPage("login");
  }, []);

  const navigateTo = useCallback(
    (target) => {
      // If in a room, allow navigating back
      if (page === "room" && target !== "room") {
        // Don't leave room, just switch view -- user stays connected
      }
      setPage(target);
    },
    [page]
  );

  const handleShardUpdate = useCallback((newBalance) => {
    if (newBalance !== undefined && newBalance !== null) {
      setShardBalance(newBalance);
    }
  }, []);

  // Login page -- full screen, no sidebar
  if (page === "login") {
    return (
      <div style={appStyle}>
        <Login onLogin={handleLogin} />
      </div>
    );
  }

  const qDots = qualityDots(connQuality);

  return (
    <div style={appStyle}>
      {/* Header */}
      <header style={headerStyle}>
        <div style={headerRight}>
          <span style={headerTitle}>{PAGE_TITLES[page] || "دوتاچی"}</span>
          {page === "room" && activeRoom && (
            <span style={{ fontSize: "0.82rem", color: COLORS.textSecondary }}>
              / {activeRoom.name}
            </span>
          )}
        </div>
        <div style={headerLeft}>
          {/* Shard balance */}
          <div
            style={shardDisplayStyle}
            onClick={() => navigateTo("shop")}
            title="فروشگاه"
            onMouseEnter={(e) => {
              e.currentTarget.style.background = `${COLORS.shard}1a`;
              e.currentTarget.style.borderColor = `${COLORS.shard}55`;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = `${COLORS.shard}12`;
              e.currentTarget.style.borderColor = `${COLORS.shard}33`;
            }}
          >
            <span style={{ fontSize: "0.9rem" }}>{"🔶"}</span>
            <span style={shardAmountStyle}>{formatShards(shardBalance)}</span>
          </div>

          {/* Ping display (when in room) */}
          {page === "room" && (
            <div style={pingDisplay(pingMs)}>
              <div style={pingDot(pingMs)} />
              {pingMs != null && pingMs >= 0 ? `${pingMs}ms` : "--"}
            </div>
          )}
          <span style={userInfoStyle}>{user?.display_name}</span>
          {page === "room" && (
            <button
              style={btnFlat}
              onClick={() => setPage("rooms")}
              onMouseEnter={(e) => { e.currentTarget.style.color = COLORS.text; }}
              onMouseLeave={(e) => { e.currentTarget.style.color = COLORS.textSecondary; }}
            >
              بازگشت به اتاق‌ها
            </button>
          )}
          <button
            style={btnFlat}
            onClick={handleLogout}
            onMouseEnter={(e) => { e.currentTarget.style.color = COLORS.error; }}
            onMouseLeave={(e) => { e.currentTarget.style.color = COLORS.textSecondary; }}
          >
            خروج
          </button>
        </div>
      </header>

      <div style={bodyStyle}>
        {/* Sidebar */}
        <nav style={sidebarStyle}>
          <div
            style={sidebarLogoStyle}
            onClick={() => navigateTo("rooms")}
            title="دوتاچی"
          >
            D
          </div>

          {NAV_ITEMS.map((item) => (
            <button
              key={item.key}
              style={navBtnStyle(page === item.key)}
              onClick={() => navigateTo(item.key)}
              onMouseEnter={() => setHoveredNav(item.key)}
              onMouseLeave={() => setHoveredNav(null)}
            >
              {item.icon}
              {hoveredNav === item.key && <div style={navTooltip}>{item.label}</div>}
            </button>
          ))}

          <div style={sidebarSpacer} />

          {/* Back to room indicator if user is in a room but viewing another page */}
          {activeRoom && page !== "room" && (
            <button
              style={{
                ...navBtnStyle(false),
                background: `${COLORS.success}18`,
                color: COLORS.success,
                animation: "pulse 2s ease-in-out infinite",
              }}
              onClick={() => setPage("room")}
              onMouseEnter={() => setHoveredNav("back-room")}
              onMouseLeave={() => setHoveredNav(null)}
              title="بازگشت به اتاق"
            >
              &#9664;
              {hoveredNav === "back-room" && (
                <div style={navTooltip}>بازگشت به اتاق</div>
              )}
            </button>
          )}
        </nav>

        {/* Main content */}
        <main style={mainStyle}>
          {page === "rooms" && (
            <Rooms
              onJoin={handleJoin}
              shardBalance={shardBalance}
              onShardUpdate={handleShardUpdate}
              onNavigateShop={() => navigateTo("shop")}
            />
          )}
          {page === "favorites" && (
            <Rooms
              onJoin={handleJoin}
              shardBalance={shardBalance}
              onShardUpdate={handleShardUpdate}
              onNavigateShop={() => navigateTo("shop")}
            />
          )}
          {page === "room" && (
            <Room
              room={activeRoom}
              vpnCreds={vpnCreds}
              userId={user?.user_id}
              onLeave={handleLeave}
              onToggleFavorite={toggleFavorite}
              isFavorite={favorites.includes(activeRoom?.id)}
              shardBalance={shardBalance}
              onShardUpdate={handleShardUpdate}
            />
          )}
          {page === "profile" && (
            <Profile
              user={user}
              shardBalance={shardBalance}
              onNavigateShop={() => navigateTo("shop")}
              onShardUpdate={handleShardUpdate}
            />
          )}
          {page === "shop" && <Shop />}
          {page === "guides" && <GameGuides />}
          {page === "settings" && <Settings />}
        </main>
      </div>

      {/* Footer */}
      <footer style={footerStyle}>
        <span style={{ direction: "ltr" }}>Dotachi v0.1.0</span>
        {page === "room" && connQuality && (
          <div style={qualityIndicator(connQuality)}>
            <span style={{ display: "flex", gap: "2px" }}>
              {[1, 2, 3, 4].map((i) => (
                <div
                  key={i}
                  style={{
                    width: "3px",
                    height: `${4 + i * 3}px`,
                    borderRadius: "1px",
                    background: i <= qDots.level ? qDots.color : "#333355",
                    transition: "background 0.3s ease",
                  }}
                />
              ))}
            </span>
            <span>{QUALITY_LABELS[connQuality] || connQuality}</span>
          </div>
        )}
        <span style={{ color: "#444466" }}>شبکه بازی لن</span>
      </footer>
    </div>
  );
}
