import React, { useState, useEffect, useCallback, useRef } from "react";
import { ListRooms, JoinRoom, GetRoom, PingServer, PurchaseRoom } from "../api";
import RoomCard from "../components/RoomCard";
import SearchBar from "../components/SearchBar";
import GameFilter, { GAMES } from "../components/GameFilter";

const SHARD_COLOR = "#ff9800";

// --- Filter definitions ---
const ACCESS_FILTERS = [
  { label: "همه", isPrivate: null, hasSlots: null },
  { label: "عمومی", isPrivate: false, hasSlots: null },
  { label: "خصوصی", isPrivate: true, hasSlots: null },
  { label: "جای خالی", isPrivate: null, hasSlots: true },
];

const SORT_OPTIONS = [
  { key: "newest", label: "جدیدترین" },
  { key: "players", label: "بیشترین بازیکن" },
  { key: "name", label: "نام" },
];

const DURATION_OPTIONS = [
  { key: "weekly", label: "۷ روز", discount: 0, days: 7 },
  { key: "monthly", label: "۱ ماه (۱۰٪ تخفیف)", discount: 0.1, days: 30 },
  { key: "quarterly", label: "۳ ماه (۲۵٪ تخفیف)", discount: 0.25, days: 90 },
  { key: "yearly", label: "سالانه (۴۰٪ تخفیف)", discount: 0.4, days: 365 },
];

const GAME_TAG_OPTIONS = GAMES.filter((g) => g.key !== "all");

// --- Styles ---
const pageStyle = {
  display: "flex",
  flexDirection: "column",
  gap: "16px",
  height: "100%",
  animation: "fadeIn 0.3s ease-out",
  direction: "rtl",
  textAlign: "right",
};

const topBarStyle = {
  display: "flex",
  gap: "12px",
  alignItems: "center",
  flexWrap: "wrap",
};

const filterRowStyle = {
  display: "flex",
  gap: "8px",
  alignItems: "center",
  flexWrap: "wrap",
};

const pillBtn = (active) => ({
  padding: "6px 14px",
  borderRadius: "20px",
  border: active ? "1px solid #7c4dff" : "1px solid #2a2a45",
  background: active ? "#7c4dff" : "transparent",
  color: active ? "#fff" : "#8888aa",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 500,
  transition: "all 0.2s ease",
  outline: "none",
  fontFamily: "'Vazirmatn', sans-serif",
});

const sortSelect = {
  padding: "6px 12px",
  borderRadius: "8px",
  border: "1px solid #2a2a45",
  background: "#141428",
  color: "#8888aa",
  fontSize: "0.82rem",
  outline: "none",
  cursor: "pointer",
  fontFamily: "'Vazirmatn', sans-serif",
  direction: "rtl",
};

const buyRoomBtn = {
  marginRight: "auto",
  padding: "8px 20px",
  borderRadius: "20px",
  border: `1px solid ${SHARD_COLOR}`,
  background: `linear-gradient(135deg, ${SHARD_COLOR}18, ${SHARD_COLOR}08)`,
  color: SHARD_COLOR,
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 600,
  transition: "all 0.2s ease",
  letterSpacing: "0.3px",
  fontFamily: "'Vazirmatn', sans-serif",
};

const gridStyle = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))",
  gap: "16px",
  flex: 1,
  overflowY: "auto",
  paddingBottom: "16px",
};

const emptyStyle = {
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  flex: 1,
  color: "#555577",
  gap: "12px",
};

const emptyIcon = {
  fontSize: "2.5rem",
  opacity: 0.4,
};

const modalOverlay = {
  position: "fixed",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background: "rgba(0, 0, 0, 0.7)",
  display: "flex",
  justifyContent: "center",
  alignItems: "center",
  zIndex: 1000,
  backdropFilter: "blur(4px)",
};

const modalBox = {
  background: "#141428",
  borderRadius: "16px",
  padding: "32px",
  width: "100%",
  maxWidth: "480px",
  boxShadow: "0 16px 48px rgba(0, 0, 0, 0.6)",
  border: "1px solid #2a2a45",
  animation: "fadeIn 0.2s ease-out",
  maxHeight: "90vh",
  overflowY: "auto",
  direction: "rtl",
  textAlign: "right",
};

const modalTitle = {
  fontSize: "1.1rem",
  fontWeight: 600,
  marginBottom: "16px",
  color: "#e8e8f0",
};

const modalInput = {
  width: "100%",
  padding: "12px 16px",
  marginBottom: "14px",
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

const modalBtnRow = {
  display: "flex",
  gap: "10px",
  justifyContent: "flex-start",
};

const modalBtnPrimary = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "none",
  background: "#7c4dff",
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.9rem",
  fontFamily: "'Vazirmatn', sans-serif",
};

const modalBtnCancel = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "transparent",
  color: "#8888aa",
  cursor: "pointer",
  fontSize: "0.9rem",
  fontFamily: "'Vazirmatn', sans-serif",
};

const modalLabel = {
  fontSize: "0.82rem",
  color: "#8888aa",
  marginBottom: "6px",
  fontWeight: 500,
};

const modalSelect = {
  width: "100%",
  padding: "10px 14px",
  marginBottom: "14px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.95rem",
  outline: "none",
  cursor: "pointer",
  fontFamily: "'Vazirmatn', sans-serif",
  direction: "rtl",
};

const sliderRow = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
  marginBottom: "14px",
};

const sliderStyle = {
  flex: 1,
  accentColor: SHARD_COLOR,
  cursor: "pointer",
};

const sliderValue = {
  fontSize: "1rem",
  fontWeight: 700,
  color: SHARD_COLOR,
  fontFamily: "monospace",
  minWidth: "40px",
  textAlign: "left",
  direction: "ltr",
};

const priceBox = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px",
  border: "1px solid #2a2a45",
  marginBottom: "14px",
  textAlign: "center",
};

const priceAmount = {
  fontSize: "1.3rem",
  fontWeight: 700,
  color: SHARD_COLOR,
  fontFamily: "monospace",
  direction: "ltr",
};

const priceLabel = {
  fontSize: "0.8rem",
  color: "#8888aa",
  marginTop: "4px",
};

const balanceRow = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginBottom: "16px",
  padding: "10px 14px",
  background: "#0d0d20",
  borderRadius: "8px",
  border: "1px solid #1a1a30",
};

const toggleRow = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  marginBottom: "14px",
};

const toggleSwitch = (on) => ({
  width: "40px",
  height: "22px",
  borderRadius: "11px",
  background: on ? SHARD_COLOR : "#2a2a45",
  cursor: "pointer",
  position: "relative",
  transition: "background 0.2s ease",
  border: "none",
  padding: 0,
});

const toggleKnob = (on) => ({
  width: "16px",
  height: "16px",
  borderRadius: "50%",
  background: "#fff",
  position: "absolute",
  top: "3px",
  right: on ? "21px" : "3px",
  transition: "right 0.2s ease",
});

const paginationStyle = {
  display: "flex",
  justifyContent: "center",
  gap: "8px",
  paddingBottom: "8px",
};

const pageBtn = (active) => ({
  width: "32px",
  height: "32px",
  borderRadius: "8px",
  border: active ? "1px solid #7c4dff" : "1px solid #2a2a45",
  background: active ? "#7c4dff" : "#141428",
  color: active ? "#fff" : "#8888aa",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 500,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
});

const loadingStyle = {
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  flex: 1,
  color: "#8888aa",
  fontSize: "0.95rem",
  gap: "10px",
};

const spinnerSmall = {
  display: "inline-block",
  width: "18px",
  height: "18px",
  border: "2px solid #7c4dff33",
  borderTopColor: "#7c4dff",
  borderRadius: "50%",
  animation: "spin 0.6s linear infinite",
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
  display: "flex",
  justifyContent: "center",
  alignItems: "center",
  gap: "12px",
  flexWrap: "wrap",
};

const retryBtn = {
  background: "none",
  border: "1px solid #ff5252",
  color: "#ff5252",
  borderRadius: "8px",
  padding: "6px 16px",
  cursor: "pointer",
  fontSize: "0.82rem",
  fontWeight: 500,
  fontFamily: "'Vazirmatn', sans-serif",
};

function detectGameKey(name) {
  const lower = (name || "").toLowerCase();
  if (lower.includes("dota")) return "dota2";
  if (lower.includes("cs2") || lower.includes("counter")) return "cs2";
  if (lower.includes("wc3") || lower.includes("warcraft")) return "wc3";
  if (lower.includes("aoe") || lower.includes("age of")) return "aoe";
  if (lower.includes("valorant") || lower.includes("val")) return "valorant";
  if (lower.includes("minecraft") || lower.includes("mc")) return "mc";
  return "other";
}

function formatShards(n) {
  return n.toLocaleString("en-US");
}

function calcPrice(slots, duration) {
  const basePerDay = slots * 1000;
  if (duration === "yearly") return Math.round(basePerDay * 365 * 0.6);
  if (duration === "quarterly") return Math.round(basePerDay * 90 * 0.75);
  if (duration === "monthly") return Math.round(basePerDay * 30 * 0.9);
  // weekly (7 days, minimum)
  return basePerDay * 7;
}

export default function RoomsPage({ onJoin, shardBalance, onShardUpdate, onNavigateShop }) {
  const [rooms, setRooms] = useState([]);
  const [query, setQuery] = useState("");
  const [filterIdx, setFilterIdx] = useState(0);
  const [gameFilter, setGameFilter] = useState("all");
  const [sortKey, setSortKey] = useState("newest");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [offline, setOffline] = useState(false);
  const [passwordModal, setPasswordModal] = useState(null);
  const [roomPassword, setRoomPassword] = useState("");
  const [joining, setJoining] = useState(false);
  const [currentPage, setCurrentPage] = useState(0);
  const [favorites, setFavorites] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem("dotachi_favorites") || "[]");
    } catch {
      return [];
    }
  });
  const [roomPings, setRoomPings] = useState({});

  // Purchase modal state
  const [showPurchase, setShowPurchase] = useState(false);
  const [purchaseName, setPurchaseName] = useState("");
  const [purchaseGameTag, setPurchaseGameTag] = useState("dota2");
  const [purchaseSlots, setPurchaseSlots] = useState(15);
  const [purchaseDuration, setPurchaseDuration] = useState("weekly");
  const [purchasePrivate, setPurchasePrivate] = useState(false);
  const [purchasePassword, setPurchasePassword] = useState("");
  const [purchasing, setPurchasing] = useState(false);
  const [purchaseError, setPurchaseError] = useState("");

  const debounceRef = useRef(null);

  // Save favorites to localStorage
  useEffect(() => {
    try {
      localStorage.setItem("dotachi_favorites", JSON.stringify(favorites));
    } catch {
      // ignore
    }
  }, [favorites]);

  const fetchRooms = useCallback(
    async (q, fi, page) => {
      setLoading(true);
      setError("");
      setOffline(false);
      const f = ACCESS_FILTERS[fi];
      try {
        const isPrivateVal = f.isPrivate === null ? null : f.isPrivate;
        const hasSlotsVal = f.hasSlots === null ? null : f.hasSlots;
        const data = await ListRooms(q, isPrivateVal, hasSlotsVal, page);
        setRooms(data || []);
        // Ping each room's node in background
        pingRooms(data || []);
      } catch (err) {
        const errStr = String(err);
        if (errStr.includes("fetch") || errStr.includes("network") || errStr.includes("Failed") || errStr.includes("ECONNREFUSED")) {
          setOffline(true);
        } else {
          setError(errStr);
        }
        setRooms([]);
      } finally {
        setLoading(false);
      }
    },
    []
  );

  // Ping rooms' nodes
  const pingRooms = useCallback(async (roomList) => {
    const hosts = new Set();
    const roomHostMap = {};
    for (const r of roomList) {
      if (r.node_name) {
        hosts.add(r.node_name);
        roomHostMap[r.id] = r.node_name;
      }
    }
    const hostPings = {};
    for (const host of hosts) {
      try {
        const ms = await PingServer(host);
        hostPings[host] = ms;
      } catch {
        hostPings[host] = -1;
      }
    }
    const pings = {};
    for (const r of roomList) {
      if (roomHostMap[r.id] && hostPings[roomHostMap[r.id]] !== undefined) {
        pings[r.id] = hostPings[roomHostMap[r.id]];
      }
    }
    setRoomPings(pings);
  }, []);

  // Initial load and on filter change
  useEffect(() => {
    fetchRooms(query, filterIdx, currentPage);
  }, [filterIdx, currentPage]); // eslint-disable-line react-hooks/exhaustive-deps

  // Debounced search
  const handleSearch = useCallback(
    (val) => {
      setQuery(val);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        setCurrentPage(0);
        fetchRooms(val, filterIdx, 0);
      }, 350);
    },
    [fetchRooms, filterIdx]
  );

  const handleJoinClick = async (room) => {
    if (room.is_private) {
      setPasswordModal(room);
      setRoomPassword("");
      return;
    }
    await doJoin(room, "");
  };

  const doJoin = async (room, pw) => {
    setJoining(true);
    setError("");
    try {
      const creds = await JoinRoom(room.id, pw);
      const updated = await GetRoom(room.id);
      setPasswordModal(null);
      onJoin(updated, creds);
    } catch (err) {
      setError(String(err));
    } finally {
      setJoining(false);
    }
  };

  const toggleFavorite = (roomId) => {
    setFavorites((prev) =>
      prev.includes(roomId)
        ? prev.filter((id) => id !== roomId)
        : [...prev, roomId]
    );
  };

  // Purchase handlers
  const purchasePrice = calcPrice(purchaseSlots, purchaseDuration);
  const canAfford = (shardBalance || 0) >= purchasePrice;
  const purchaseDurationDays = (DURATION_OPTIONS.find(d => d.key === purchaseDuration) || {}).days || 7;

  const handlePurchaseOpen = () => {
    setShowPurchase(true);
    setPurchaseName("");
    setPurchaseGameTag("dota2");
    setPurchaseSlots(15);
    setPurchaseDuration("weekly");
    setPurchasePrivate(false);
    setPurchasePassword("");
    setPurchaseError("");
  };

  const handlePurchaseSubmit = async () => {
    if (!purchaseName.trim()) {
      setPurchaseError("نام اتاق الزامی است.");
      return;
    }
    if (!canAfford) {
      setPurchaseError("شارد کافی نیست!");
      return;
    }
    setPurchasing(true);
    setPurchaseError("");
    try {
      const result = await PurchaseRoom(
        purchaseName,
        purchaseGameTag,
        purchaseSlots,
        purchaseDuration,
        purchaseDurationDays,
        purchasePrivate,
        purchasePrivate ? purchasePassword : ""
      );
      // Update shard balance
      if (result && result.new_balance !== undefined) {
        onShardUpdate(result.new_balance);
      }
      setShowPurchase(false);
      // Refresh rooms
      fetchRooms(query, filterIdx, currentPage);
    } catch (err) {
      setPurchaseError(String(err));
    } finally {
      setPurchasing(false);
    }
  };

  // Filter and sort rooms
  let displayRooms = [...rooms];

  // Game filter
  if (gameFilter !== "all") {
    displayRooms = displayRooms.filter(
      (r) => detectGameKey(r.name) === gameFilter || r.game_tag === gameFilter
    );
  }

  // Sort
  if (sortKey === "players") {
    displayRooms.sort((a, b) => b.current_players - a.current_players);
  } else if (sortKey === "name") {
    displayRooms.sort((a, b) => a.name.localeCompare(b.name));
  }
  // "newest" = default server order (already sorted by created_at desc)

  return (
    <div style={pageStyle}>
      {/* Offline banner */}
      {offline && (
        <div style={offlineBanner}>
          <span>سرور در دسترس نیست. تلگرام: @coded_pro</span>
          <button
            style={retryBtn}
            onClick={() => fetchRooms(query, filterIdx, currentPage)}
          >
            تلاش مجدد
          </button>
        </div>
      )}

      {/* Search + game filter */}
      <div style={topBarStyle}>
        <div style={{ flex: 1, minWidth: "200px" }}>
          <SearchBar value={query} onChange={handleSearch} />
        </div>
      </div>

      <GameFilter active={gameFilter} onChange={setGameFilter} />

      {/* Access filters + sort + buy room */}
      <div style={filterRowStyle}>
        {ACCESS_FILTERS.map((f, i) => (
          <button
            key={f.label}
            style={pillBtn(i === filterIdx)}
            onClick={() => {
              setFilterIdx(i);
              setCurrentPage(0);
            }}
          >
            {f.label}
          </button>
        ))}

        <select
          style={sortSelect}
          value={sortKey}
          onChange={(e) => setSortKey(e.target.value)}
        >
          {SORT_OPTIONS.map((s) => (
            <option key={s.key} value={s.key}>
              {s.label}
            </option>
          ))}
        </select>

        <button
          style={buyRoomBtn}
          onClick={handlePurchaseOpen}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = `linear-gradient(135deg, ${SHARD_COLOR}28, ${SHARD_COLOR}18)`;
            e.currentTarget.style.boxShadow = `0 4px 16px ${SHARD_COLOR}22`;
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = `linear-gradient(135deg, ${SHARD_COLOR}18, ${SHARD_COLOR}08)`;
            e.currentTarget.style.boxShadow = "none";
          }}
        >
          {"🔶"} خرید اتاق
        </button>
      </div>

      {error && (
        <div
          style={{
            color: "#ff5252",
            fontSize: "0.85rem",
            padding: "10px 14px",
            background: "#ff525212",
            borderRadius: "8px",
            border: "1px solid #ff525222",
          }}
        >
          {error}
        </div>
      )}

      {loading && (
        <div style={loadingStyle}>
          <span style={spinnerSmall} />
          در حال بارگذاری اتاق‌ها...
        </div>
      )}

      {!loading && displayRooms.length === 0 && !offline && (
        <div style={emptyStyle}>
          <span style={emptyIcon}>&#9744;</span>
          <span style={{ fontSize: "1rem" }}>اتاقی یافت نشد</span>
          <span style={{ fontSize: "0.85rem", color: "#444466" }}>
            فیلترها یا عبارت جستجو را تغییر دهید
          </span>
        </div>
      )}

      {!loading && displayRooms.length > 0 && (
        <div style={gridStyle}>
          {displayRooms.map((room) => (
            <RoomCard
              key={room.id}
              room={room}
              onJoin={() => handleJoinClick(room)}
              pingMs={roomPings[room.id]}
              isFavorite={favorites.includes(room.id)}
              onToggleFavorite={toggleFavorite}
            />
          ))}
        </div>
      )}

      {/* Pagination */}
      {!loading && rooms.length > 0 && (
        <div style={paginationStyle}>
          {currentPage > 0 && (
            <button
              style={pageBtn(false)}
              onClick={() => setCurrentPage((p) => p - 1)}
            >
              &gt;
            </button>
          )}
          <button style={pageBtn(true)}>{currentPage + 1}</button>
          {rooms.length >= 20 && (
            <button
              style={pageBtn(false)}
              onClick={() => setCurrentPage((p) => p + 1)}
            >
              &lt;
            </button>
          )}
        </div>
      )}

      {/* Password modal */}
      {passwordModal && (
        <div style={modalOverlay} onClick={() => setPasswordModal(null)}>
          <div style={modalBox} onClick={(e) => e.stopPropagation()}>
            <div style={modalTitle}>
              رمز اتاق "{passwordModal.name}" را وارد کنید
            </div>
            <input
              style={modalInput}
              type="password"
              placeholder="رمز اتاق"
              value={roomPassword}
              onChange={(e) => setRoomPassword(e.target.value)}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") doJoin(passwordModal, roomPassword);
              }}
            />
            {error && (
              <div
                style={{
                  color: "#ff5252",
                  marginBottom: "12px",
                  fontSize: "0.85rem",
                }}
              >
                {error}
              </div>
            )}
            <div style={modalBtnRow}>
              <button
                style={modalBtnCancel}
                onClick={() => setPasswordModal(null)}
              >
                انصراف
              </button>
              <button
                style={modalBtnPrimary}
                disabled={joining}
                onClick={() => doJoin(passwordModal, roomPassword)}
              >
                {joining ? "در حال ورود..." : "ورود"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Purchase Room Modal */}
      {showPurchase && (
        <div style={modalOverlay} onClick={() => setShowPurchase(false)}>
          <div style={modalBox} onClick={(e) => e.stopPropagation()}>
            <div style={modalTitle}>{"🔶"} خرید اتاق</div>

            {/* Balance display */}
            <div style={balanceRow}>
              <span style={{ color: "#8888aa", fontSize: "0.85rem" }}>موجودی شما</span>
              <span style={{ color: SHARD_COLOR, fontWeight: 700, fontFamily: "monospace", direction: "ltr" }}>
                {"🔶"} {formatShards(shardBalance || 0)}
              </span>
            </div>

            {/* Room name */}
            <div style={modalLabel}>نام اتاق</div>
            <input
              style={modalInput}
              type="text"
              placeholder="اتاق دوتای من"
              value={purchaseName}
              onChange={(e) => setPurchaseName(e.target.value)}
              autoFocus
            />

            {/* Game tag */}
            <div style={modalLabel}>بازی</div>
            <select
              style={modalSelect}
              value={purchaseGameTag}
              onChange={(e) => setPurchaseGameTag(e.target.value)}
            >
              {GAME_TAG_OPTIONS.map((g) => (
                <option key={g.key} value={g.key}>{g.label}</option>
              ))}
            </select>

            {/* Slots */}
            <div style={modalLabel}>ظرفیت (بازیکن)</div>
            <div style={sliderRow}>
              <input
                type="range"
                min="5"
                max="100"
                step="5"
                value={purchaseSlots}
                onChange={(e) => setPurchaseSlots(Number(e.target.value))}
                style={sliderStyle}
              />
              <span style={sliderValue}>{purchaseSlots}</span>
            </div>

            {/* Duration */}
            <div style={modalLabel}>مدت زمان</div>
            <select
              style={modalSelect}
              value={purchaseDuration}
              onChange={(e) => setPurchaseDuration(e.target.value)}
            >
              {DURATION_OPTIONS.map((d) => (
                <option key={d.key} value={d.key}>{d.label}</option>
              ))}
            </select>

            {/* Private toggle */}
            <div style={toggleRow}>
              <span style={{ color: "#e8e8f0", fontSize: "0.9rem" }}>اتاق خصوصی</span>
              <button
                style={toggleSwitch(purchasePrivate)}
                onClick={() => setPurchasePrivate(!purchasePrivate)}
              >
                <div style={toggleKnob(purchasePrivate)} />
              </button>
            </div>

            {/* Password (only if private) */}
            {purchasePrivate && (
              <>
                <div style={modalLabel}>رمز اتاق</div>
                <input
                  style={modalInput}
                  type="password"
                  placeholder="رمز عبور برای اتاق"
                  value={purchasePassword}
                  onChange={(e) => setPurchasePassword(e.target.value)}
                />
              </>
            )}

            {/* Price display */}
            <div style={priceBox}>
              <div style={priceAmount}>{"🔶"} {formatShards(purchasePrice)} شارد</div>
              <div style={priceLabel}>
                {purchaseSlots} ظرفیت x {purchaseDurationDays} روز
                {purchaseDuration === "monthly" ? " (۱۰٪ تخفیف)" :
                 purchaseDuration === "quarterly" ? " (۲۵٪ تخفیف)" :
                 purchaseDuration === "yearly" ? " (۴۰٪ تخفیف)" : ""}
              </div>
              {!canAfford && (
                <div style={{ color: "#ff5252", fontSize: "0.82rem", marginTop: "8px" }}>
                  شارد کافی نیست!{" "}
                  <span
                    style={{ color: SHARD_COLOR, cursor: "pointer", textDecoration: "underline" }}
                    onClick={() => {
                      setShowPurchase(false);
                      if (onNavigateShop) onNavigateShop();
                    }}
                  >
                    خرید شارد
                  </span>
                </div>
              )}
            </div>

            {purchaseError && (
              <div style={{ color: "#ff5252", fontSize: "0.85rem", marginBottom: "12px", textAlign: "center" }}>
                {purchaseError}
              </div>
            )}

            <div style={modalBtnRow}>
              <button style={modalBtnCancel} onClick={() => setShowPurchase(false)}>
                انصراف
              </button>
              <button
                style={{
                  ...modalBtnPrimary,
                  background: canAfford
                    ? `linear-gradient(135deg, ${SHARD_COLOR}, #e68900)`
                    : "#2a2a45",
                  color: canAfford ? "#fff" : "#666",
                  cursor: canAfford && !purchasing ? "pointer" : "not-allowed",
                  boxShadow: canAfford ? `0 4px 12px ${SHARD_COLOR}33` : "none",
                }}
                disabled={!canAfford || purchasing}
                onClick={handlePurchaseSubmit}
              >
                {purchasing ? "در حال خرید..." : "خرید"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
