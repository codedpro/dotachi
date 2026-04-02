import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  LeaveRoom,
  GetMembers,
  ConnectVPN,
  StopVPN,
  GetVPNStatus,
  GetPingStats,
  GetConnectionQuality,
  ExtendRoom,
  SetRoomRole,
  TransferRoom,
  CreateInvite,
} from "../api";
import ConnectionStatus from "../components/ConnectionStatus";
import PingGraph from "../components/PingGraph";
import LocalIP from "../components/LocalIP";
import RoomChat from "../components/RoomChat";

const SHARD_COLOR = "#ff9800";

// --- Styles ---
const containerStyle = {
  display: "flex",
  gap: "20px",
  height: "100%",
  animation: "fadeIn 0.3s ease-out",
  direction: "rtl",
  textAlign: "right",
};

const rightCol = {
  flex: "1 1 55%",
  display: "flex",
  flexDirection: "column",
  gap: "16px",
  overflowY: "auto",
};

const leftCol = {
  flex: "1 1 45%",
  display: "flex",
  flexDirection: "column",
  gap: "16px",
  overflowY: "auto",
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

const roomTitleStyle = {
  fontSize: "1.4rem",
  fontWeight: 700,
  color: "#e8e8f0",
  marginBottom: "4px",
  display: "flex",
  alignItems: "center",
  gap: "10px",
};

const infoRow = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "8px 0",
  borderBottom: "1px solid #1a1a30",
};

const labelText = { color: "#8888aa", fontSize: "0.9rem" };
const valueText = { color: "#e8e8f0", fontWeight: 500, fontSize: "0.9rem" };

const memberItem = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "10px 0",
  borderBottom: "1px solid #1a1a30",
};

const memberName = {
  color: "#e8e8f0",
  fontSize: "0.95rem",
  fontWeight: 500,
  display: "flex",
  alignItems: "center",
  gap: "8px",
};

const onlineDot = {
  width: "8px",
  height: "8px",
  borderRadius: "50%",
  background: "#00e676",
  boxShadow: "0 0 6px #00e67666",
  flexShrink: 0,
};

const joinedAtStyle = {
  color: "#555577",
  fontSize: "0.8rem",
  fontFamily: "monospace",
  direction: "ltr",
};

const ROLE_LABELS = {
  owner: "مالک",
  admin: "مدیر",
  member: "عضو",
};

const roleBadge = (role) => {
  const configs = {
    owner: { bg: `${SHARD_COLOR}18`, color: SHARD_COLOR, border: `${SHARD_COLOR}33` },
    admin: { bg: "#7c4dff18", color: "#7c4dff", border: "#7c4dff33" },
    member: { bg: "#8888aa12", color: "#8888aa", border: "#8888aa22" },
  };
  const c = configs[role] || configs.member;
  return {
    padding: "2px 8px",
    borderRadius: "8px",
    fontSize: "0.7rem",
    fontWeight: 600,
    background: c.bg,
    color: c.color,
    border: `1px solid ${c.border}`,
    letterSpacing: "0.5px",
  };
};

const credCard = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px",
  border: "1px solid #2a2a45",
};

const credRow = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  padding: "8px 0",
  borderBottom: "1px solid #1a1a30",
  cursor: "pointer",
  transition: "background 0.15s ease",
  borderRadius: "4px",
  marginLeft: "-4px",
  marginRight: "-4px",
  paddingLeft: "4px",
  paddingRight: "4px",
};

const credLabel = { color: "#8888aa", fontSize: "0.82rem" };
const credValue = {
  color: "#e8e8f0",
  fontFamily: "monospace",
  fontSize: "0.88rem",
  userSelect: "all",
  direction: "ltr",
};

const copyHint = {
  fontSize: "0.7rem",
  color: "#555577",
  marginRight: "8px",
};

const statGrid = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "12px",
  marginBottom: "16px",
};

const statBox = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "14px 16px",
  border: "1px solid #1a1a30",
};

const statLabel = {
  fontSize: "0.72rem",
  color: "#8888aa",
  letterSpacing: "0.5px",
  marginBottom: "6px",
};

const statValue = (color) => ({
  fontSize: "1.3rem",
  fontWeight: 700,
  color: color || "#e8e8f0",
  fontFamily: "monospace",
  direction: "ltr",
});

const btnRow = {
  display: "flex",
  gap: "10px",
  flexWrap: "wrap",
};

const btnPrimary = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "none",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.9rem",
  transition: "all 0.2s ease",
  boxShadow: "0 4px 12px rgba(124, 77, 255, 0.2)",
  fontFamily: "'Vazirmatn', sans-serif",
};

const btnSecondary = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "transparent",
  color: "#8888aa",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.9rem",
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

const btnDanger = {
  padding: "10px 24px",
  borderRadius: "10px",
  border: "1px solid #ff5252",
  background: "transparent",
  color: "#ff5252",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.9rem",
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

const btnShard = {
  padding: "8px 18px",
  borderRadius: "10px",
  border: `1px solid ${SHARD_COLOR}`,
  background: `linear-gradient(135deg, ${SHARD_COLOR}18, ${SHARD_COLOR}08)`,
  color: SHARD_COLOR,
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.85rem",
  transition: "all 0.2s ease",
  fontFamily: "'Vazirmatn', sans-serif",
};

const errorStyle = {
  color: "#ff5252",
  fontSize: "0.85rem",
  marginTop: "8px",
  padding: "8px 12px",
  background: "#ff525212",
  borderRadius: "8px",
  border: "1px solid #ff525222",
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
  marginBottom: "12px",
};

const qualityBar = (quality) => {
  const colors = {
    excellent: "#00e676",
    good: "#7c4dff",
    fair: "#ffab00",
    poor: "#ff5252",
  };
  const widths = {
    excellent: "100%",
    good: "75%",
    fair: "50%",
    poor: "25%",
  };
  return {
    height: "100%",
    width: widths[quality] || "0%",
    background: colors[quality] || "#555",
    borderRadius: "2px",
    transition: "width 0.5s ease",
  };
};

const QUALITY_LABELS = {
  excellent: "عالی",
  good: "خوب",
  fair: "متوسط",
  poor: "ضعیف",
};

const confirmOverlay = {
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

const confirmBox = {
  background: "#141428",
  borderRadius: "16px",
  padding: "32px",
  width: "100%",
  maxWidth: "400px",
  boxShadow: "0 16px 48px rgba(0, 0, 0, 0.6)",
  border: "1px solid #2a2a45",
  textAlign: "center",
  direction: "rtl",
};

const sharedBillingCard = {
  background: `${SHARD_COLOR}08`,
  borderRadius: "10px",
  padding: "16px",
  border: `1px solid ${SHARD_COLOR}22`,
  marginBottom: "16px",
};

const memberActionBtn = {
  padding: "4px 10px",
  borderRadius: "6px",
  border: "1px solid #2a2a45",
  background: "transparent",
  color: "#8888aa",
  cursor: "pointer",
  fontSize: "0.72rem",
  fontWeight: 500,
  transition: "all 0.15s ease",
  marginRight: "4px",
};

const extendSelect = {
  padding: "8px 12px",
  borderRadius: "8px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.9rem",
  outline: "none",
  cursor: "pointer",
  marginLeft: "8px",
  fontFamily: "'Vazirmatn', sans-serif",
  direction: "rtl",
};

const inviteCard = {
  background: "#0d0d20",
  borderRadius: "10px",
  padding: "16px",
  border: "1px solid #2a2a45",
  marginTop: "12px",
};

function formatJoinedAt(iso) {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } catch {
    return "";
  }
}

function getPingColor(ms) {
  if (ms == null || ms < 0) return "#8888aa";
  if (ms < 50) return "#00e676";
  if (ms < 100) return "#ffab00";
  return "#ff5252";
}

function formatShards(n) {
  if (n == null) return "0";
  return n.toLocaleString("en-US");
}

function formatExpiryCountdown(expiresAt) {
  if (!expiresAt) return null;
  try {
    const exp = new Date(expiresAt);
    const now = new Date();
    const diff = exp - now;
    if (diff <= 0) return "منقضی شده";
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    if (days > 0) return `${days} روز ${hours} ساعت`;
    const mins = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours} ساعت ${mins} دقیقه`;
  } catch {
    return null;
  }
}

export default function RoomPage({
  room,
  vpnCreds,
  userId,
  onLeave,
  onToggleFavorite,
  isFavorite,
  shardBalance,
  onShardUpdate,
}) {
  const [members, setMembers] = useState([]);
  const [vpnStatus, setVpnStatus] = useState("disconnected");
  const [error, setError] = useState("");
  const [offline, setOffline] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const [showLeaveConfirm, setShowLeaveConfirm] = useState(false);
  const [pingStats, setPingStats] = useState({ last_ping: -1, avg_ping: -1, packet_loss: 0, jitter: 0 });
  const [quality, setQuality] = useState("--");
  const [pingHistory, setPingHistory] = useState([]);
  const [copiedField, setCopiedField] = useState(null);
  const [memberMenu, setMemberMenu] = useState(null); // user_id of member with open menu
  const [showExtend, setShowExtend] = useState(false);
  const [extendDuration, setExtendDuration] = useState("weekly");
  const [extending, setExtending] = useState(false);
  const [showTransfer, setShowTransfer] = useState(false);
  const [transferTarget, setTransferTarget] = useState(null);
  const [transferring, setTransferring] = useState(false);
  const [sharedSpent, setSharedSpent] = useState(0);
  const [connectedSeconds, setConnectedSeconds] = useState(0);
  const [inviteLink, setInviteLink] = useState("");
  const [inviteCopied, setInviteCopied] = useState(false);
  const [creatingInvite, setCreatingInvite] = useState(false);
  const pollRef = useRef(null);
  const sharedTimerRef = useRef(null);
  const inviteCopyTimerRef = useRef(null);

  const isShared = !room?.is_private && !room?.owner_id;
  const isOwner = room?.owner_id === userId;
  const userRole = members.find((m) => m.user_id === userId)?.role;
  const canManage = isOwner || userRole === "admin" || userRole === "owner";

  const fetchMembers = useCallback(async () => {
    if (!room) return;
    try {
      const m = await GetMembers(room.id);
      if (m) setMembers(m);
      setOffline(false);
    } catch (err) {
      const errStr = String(err);
      if (errStr.includes("fetch") || errStr.includes("network") || errStr.includes("Failed") || errStr.includes("ECONNREFUSED")) {
        setOffline(true);
      }
    }
  }, [room]);

  const pollStatus = useCallback(async () => {
    try {
      const s = await GetVPNStatus();
      setVpnStatus(s);
    } catch {
      // ignore
    }
    try {
      const ps = await GetPingStats();
      if (ps) {
        setPingStats(ps);
        setPingHistory((prev) => {
          const next = [...prev, ps.last_ping];
          return next.slice(-20);
        });
      }
    } catch {
      // ignore
    }
    try {
      const q = await GetConnectionQuality();
      if (q) setQuality(q);
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    fetchMembers();
    if (vpnCreds?.vpn_host) {
      handleConnect();
    }
    pollRef.current = setInterval(() => {
      pollStatus();
      fetchMembers();
    }, 3000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Shared LAN billing timer
  useEffect(() => {
    if (!isShared) return;
    const startTime = Date.now();
    sharedTimerRef.current = setInterval(() => {
      const elapsed = (Date.now() - startTime) / 1000;
      setConnectedSeconds(Math.floor(elapsed));
      // 2000 shards/hour = 0.5556 shards/second
      setSharedSpent(Math.round((elapsed / 3600) * 2000));
    }, 1000);
    return () => {
      if (sharedTimerRef.current) clearInterval(sharedTimerRef.current);
    };
  }, [isShared]);

  const handleConnect = async () => {
    if (!vpnCreds) return;
    setError("");
    setVpnStatus("connecting");
    try {
      await ConnectVPN(
        vpnCreds.vpn_host,
        vpnCreds.hub,
        vpnCreds.vpn_username,
        vpnCreds.vpn_password,
        vpnCreds.subnet || ""
      );
    } catch (err) {
      setError(String(err));
      setVpnStatus("disconnected");
    }
  };

  const handleDisconnect = async () => {
    setError("");
    try {
      await StopVPN();
      setVpnStatus("disconnected");
      setPingHistory([]);
      setPingStats({ last_ping: -1, avg_ping: -1, packet_loss: 0, jitter: 0 });
      setQuality("--");
    } catch (err) {
      setError(String(err));
    }
  };

  const handleLeave = async () => {
    if (!room) return;
    setLeaving(true);
    setError("");
    setShowLeaveConfirm(false);
    try {
      await StopVPN();
      await LeaveRoom(room.id);
      onLeave();
    } catch (err) {
      setError(String(err));
      setLeaving(false);
    }
  };

  const handleCopy = (text, field) => {
    try {
      navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 1500);
    } catch {
      // fallback
    }
  };

  const handleSetRole = async (targetUserId, role) => {
    try {
      await SetRoomRole(room.id, targetUserId, role);
      setMemberMenu(null);
      fetchMembers();
    } catch (err) {
      setError(String(err));
    }
  };

  const handleTransferOwnership = async () => {
    if (!transferTarget) return;
    setTransferring(true);
    try {
      await TransferRoom(room.id, transferTarget);
      setShowTransfer(false);
      setTransferTarget(null);
      fetchMembers();
    } catch (err) {
      setError(String(err));
    } finally {
      setTransferring(false);
    }
  };

  const handleExtend = async () => {
    setExtending(true);
    setError("");
    try {
      const daysMap = { weekly: 7, monthly: 30, quarterly: 90, yearly: 365 };
      const result = await ExtendRoom(room.id, extendDuration, daysMap[extendDuration] || 7);
      if (result && result.new_balance !== undefined && onShardUpdate) {
        onShardUpdate(result.new_balance);
      }
      setShowExtend(false);
    } catch (err) {
      setError(String(err));
    } finally {
      setExtending(false);
    }
  };

  const handleCreateInvite = async () => {
    if (!room) return;
    setCreatingInvite(true);
    try {
      const result = await CreateInvite(room.id);
      if (result && result.link) {
        setInviteLink(result.link);
      }
    } catch (err) {
      setError(String(err));
    } finally {
      setCreatingInvite(false);
    }
  };

  const handleCopyInvite = () => {
    if (!inviteLink) return;
    try {
      navigator.clipboard.writeText(inviteLink);
      setInviteCopied(true);
      if (inviteCopyTimerRef.current) clearTimeout(inviteCopyTimerRef.current);
      inviteCopyTimerRef.current = setTimeout(() => setInviteCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  const expiryText = formatExpiryCountdown(room?.expires_at);

  if (!room) return null;

  return (
    <div style={containerStyle}>
      {/* Right column (RTL: first in DOM = right side): Room info + Members */}
      <div style={rightCol}>
        {offline && (
          <div style={offlineBanner}>
            سرور در دسترس نیست. تلگرام: @coded_pro
          </div>
        )}

        <div style={card}>
          <div style={roomTitleStyle}>
            {room.is_private && (
              <span style={{ color: "#ffab00", fontSize: "1rem" }}>&#128274;</span>
            )}
            {isShared && (
              <span style={{
                padding: "2px 10px",
                borderRadius: "8px",
                fontSize: "0.7rem",
                fontWeight: 600,
                background: "#2196f318",
                color: "#2196f3",
                border: "1px solid #2196f333",
              }}>اشتراکی</span>
            )}
            {room.name}
            {onToggleFavorite && (
              <button
                onClick={() => onToggleFavorite(room.id)}
                style={{
                  background: "none",
                  border: "none",
                  cursor: "pointer",
                  fontSize: "1.2rem",
                  color: isFavorite ? "#ffab00" : "#444466",
                  padding: "2px",
                  marginRight: "auto",
                }}
              >
                {isFavorite ? "\u2605" : "\u2606"}
              </button>
            )}
          </div>
          <div style={{ color: "#8888aa", fontSize: "0.85rem", marginBottom: "16px" }}>
            {room.node_name}
          </div>

          <div style={infoRow}>
            <span style={labelText}>{isShared ? "نوع" : "میزبان"}</span>
            <span style={valueText}>
              {isShared ? "لن اشتراکی" : room.owner_display_name}
            </span>
          </div>
          <div style={infoRow}>
            <span style={labelText}>بازیکنان</span>
            <span style={{ ...valueText, fontFamily: "monospace", direction: "ltr" }}>
              {room.current_players} / {room.max_players}
            </span>
          </div>
          <div style={infoRow}>
            <span style={labelText}>نوع</span>
            <span style={valueText}>{room.is_private ? "خصوصی" : "عمومی"}</span>
          </div>
          {expiryText && (
            <div style={infoRow}>
              <span style={labelText}>انقضا</span>
              <span style={{
                ...valueText,
                color: expiryText === "منقضی شده" ? "#ff5252"
                  : expiryText.includes("ساعت") && !expiryText.includes("روز") ? "#ff5252"
                  : SHARD_COLOR,
                fontFamily: "monospace",
              }}>
                {expiryText}
              </span>
            </div>
          )}
          <div style={infoRow}>
            <span style={labelText}>زیرشبکه</span>
            <span style={{ ...valueText, fontFamily: "monospace", direction: "ltr" }}>{room.subnet}</span>
          </div>
          <div style={{ ...infoRow, borderBottom: "none" }}>
            <span style={labelText}>سرور</span>
            <span style={valueText}>{room.node_name}</span>
          </div>

          {/* Local IP */}
          <div style={{ marginTop: "16px", padding: "12px 16px", background: "#0d0d20", borderRadius: "10px", border: "1px solid #1a1a30" }}>
            <LocalIP />
          </div>
        </div>

        {/* Shared LAN billing info */}
        {isShared && (
          <div style={{ ...card, ...sharedBillingCard }}>
            <div style={sectionTitle}>{"🔶"} صورت‌حساب لن اشتراکی</div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "12px" }}>
              <span style={{ color: "#e8e8f0", fontSize: "0.95rem" }}>نرخ</span>
              <span style={{ color: SHARD_COLOR, fontWeight: 700, fontFamily: "monospace", fontSize: "1.1rem", direction: "ltr" }}>
                {"🔶"} 2,000/ساعت
              </span>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "12px" }}>
              <span style={{ color: "#8888aa", fontSize: "0.9rem" }}>مدت اتصال</span>
              <span style={{ color: "#e8e8f0", fontFamily: "monospace", fontSize: "0.9rem", direction: "ltr" }}>
                {Math.floor(connectedSeconds / 3600)}h {Math.floor((connectedSeconds % 3600) / 60)}m {connectedSeconds % 60}s
              </span>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "12px" }}>
              <span style={{ color: "#8888aa", fontSize: "0.9rem" }}>هزینه فعلی</span>
              <span style={{ color: SHARD_COLOR, fontWeight: 600, fontFamily: "monospace", fontSize: "0.95rem", direction: "ltr" }}>
                {"🔶"} ~{formatShards(sharedSpent)}
              </span>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <span style={{ color: "#8888aa", fontSize: "0.9rem" }}>موجودی</span>
              <span style={{
                color: (shardBalance || 0) - sharedSpent > 0 ? SHARD_COLOR : "#ff5252",
                fontWeight: 600,
                fontFamily: "monospace",
                fontSize: "0.95rem",
                direction: "ltr",
              }}>
                {"🔶"} ~{formatShards(Math.max(0, (shardBalance || 0) - sharedSpent))}
              </span>
            </div>
          </div>
        )}

        {/* Room Management (owner/admin) */}
        {canManage && !isShared && (
          <div style={card}>
            <div style={sectionTitle}>مدیریت اتاق</div>
            <div style={{ display: "flex", gap: "8px", flexWrap: "wrap", marginBottom: "16px" }}>
              <button style={btnShard} onClick={() => setShowExtend(true)}>
                {"🔶"} تمدید اتاق
              </button>
              {isOwner && (
                <button
                  style={btnSecondary}
                  onClick={() => setShowTransfer(true)}
                >
                  انتقال مالکیت
                </button>
              )}
            </div>
            {/* Invite link */}
            {canManage && (
              <div style={inviteCard}>
                <div style={{ fontSize: "0.82rem", color: "#8888aa", marginBottom: "8px", fontWeight: 500 }}>لینک دعوت</div>
                {inviteLink ? (
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span
                      onClick={handleCopyInvite}
                      style={{
                        flex: 1,
                        fontFamily: "monospace",
                        fontSize: "0.82rem",
                        color: inviteCopied ? "#00e676" : "#e8e8f0",
                        cursor: "pointer",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                        direction: "ltr",
                        textAlign: "left",
                      }}
                    >
                      {inviteLink}
                    </span>
                    <button
                      style={{
                        ...memberActionBtn,
                        color: inviteCopied ? "#00e676" : "#8888aa",
                        fontFamily: "'Vazirmatn', sans-serif",
                      }}
                      onClick={handleCopyInvite}
                    >
                      {inviteCopied ? "کپی شد" : "کپی"}
                    </button>
                  </div>
                ) : (
                  <button
                    style={{
                      ...btnSecondary,
                      fontSize: "0.82rem",
                      padding: "6px 14px",
                      opacity: creatingInvite ? 0.7 : 1,
                    }}
                    onClick={handleCreateInvite}
                    disabled={creatingInvite}
                  >
                    {creatingInvite ? "در حال ایجاد..." : "ایجاد لینک دعوت"}
                  </button>
                )}
              </div>
            )}
          </div>
        )}

        {/* Members */}
        <div style={card}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "12px",
            }}
          >
            <span style={sectionTitle}>اعضا ({members.length})</span>
            <button
              style={{
                background: "none",
                border: "none",
                color: "#7c4dff",
                cursor: "pointer",
                fontSize: "0.8rem",
                fontWeight: 500,
                fontFamily: "'Vazirmatn', sans-serif",
              }}
              onClick={fetchMembers}
            >
              بروزرسانی
            </button>
          </div>
          {members.length === 0 && (
            <div style={{ color: "#555577", fontSize: "0.9rem", padding: "8px 0" }}>
              هنوز اعضایی بارگذاری نشده
            </div>
          )}
          {members.map((m) => (
            <div key={m.user_id} style={memberItem}>
              <span style={memberName}>
                <span style={onlineDot} />
                {m.display_name}
                {m.role && (
                  <span style={roleBadge(m.role)}>{ROLE_LABELS[m.role] || m.role}</span>
                )}
                {m.user_id === userId && (
                  <span
                    style={{
                      fontSize: "0.7rem",
                      color: "#7c4dff",
                      fontWeight: 600,
                      marginRight: "4px",
                    }}
                  >
                    شما
                  </span>
                )}
              </span>
              <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                <span style={joinedAtStyle}>{formatJoinedAt(m.joined_at)}</span>
                {/* Admin actions on members */}
                {canManage && m.user_id !== userId && (
                  <span style={{ position: "relative" }}>
                    <button
                      style={{
                        ...memberActionBtn,
                        color: memberMenu === m.user_id ? "#e8e8f0" : "#8888aa",
                      }}
                      onClick={() => setMemberMenu(memberMenu === m.user_id ? null : m.user_id)}
                    >
                      ...
                    </button>
                    {memberMenu === m.user_id && (
                      <div style={{
                        position: "absolute",
                        left: 0,
                        top: "100%",
                        background: "#141428",
                        border: "1px solid #2a2a45",
                        borderRadius: "8px",
                        padding: "4px",
                        zIndex: 50,
                        minWidth: "130px",
                        boxShadow: "0 8px 24px rgba(0,0,0,0.4)",
                      }}>
                        {m.role !== "admin" && (
                          <button
                            style={{
                              display: "block",
                              width: "100%",
                              padding: "6px 12px",
                              background: "none",
                              border: "none",
                              color: "#7c4dff",
                              cursor: "pointer",
                              fontSize: "0.8rem",
                              textAlign: "right",
                              borderRadius: "4px",
                              fontFamily: "'Vazirmatn', sans-serif",
                            }}
                            onClick={() => handleSetRole(m.user_id, "admin")}
                            onMouseEnter={(e) => { e.currentTarget.style.background = "#7c4dff12"; }}
                            onMouseLeave={(e) => { e.currentTarget.style.background = "none"; }}
                          >
                            تنظیم مدیر
                          </button>
                        )}
                        {m.role === "admin" && (
                          <button
                            style={{
                              display: "block",
                              width: "100%",
                              padding: "6px 12px",
                              background: "none",
                              border: "none",
                              color: "#8888aa",
                              cursor: "pointer",
                              fontSize: "0.8rem",
                              textAlign: "right",
                              borderRadius: "4px",
                              fontFamily: "'Vazirmatn', sans-serif",
                            }}
                            onClick={() => handleSetRole(m.user_id, "member")}
                            onMouseEnter={(e) => { e.currentTarget.style.background = "#8888aa12"; }}
                            onMouseLeave={(e) => { e.currentTarget.style.background = "none"; }}
                          >
                            حذف مدیر
                          </button>
                        )}
                        <button
                          style={{
                            display: "block",
                            width: "100%",
                            padding: "6px 12px",
                            background: "none",
                            border: "none",
                            color: "#ff5252",
                            cursor: "pointer",
                            fontSize: "0.8rem",
                            textAlign: "right",
                            borderRadius: "4px",
                            fontFamily: "'Vazirmatn', sans-serif",
                          }}
                          onClick={() => handleSetRole(m.user_id, "kicked")}
                          onMouseEnter={(e) => { e.currentTarget.style.background = "#ff525212"; }}
                          onMouseLeave={(e) => { e.currentTarget.style.background = "none"; }}
                        >
                          اخراج
                        </button>
                        <button
                          style={{
                            display: "block",
                            width: "100%",
                            padding: "6px 12px",
                            background: "none",
                            border: "none",
                            color: "#ff5252",
                            cursor: "pointer",
                            fontSize: "0.8rem",
                            textAlign: "right",
                            borderRadius: "4px",
                            fontFamily: "'Vazirmatn', sans-serif",
                          }}
                          onClick={() => handleSetRole(m.user_id, "banned")}
                          onMouseEnter={(e) => { e.currentTarget.style.background = "#ff525212"; }}
                          onMouseLeave={(e) => { e.currentTarget.style.background = "none"; }}
                        >
                          مسدود
                        </button>
                      </div>
                    )}
                  </span>
                )}
              </span>
            </div>
          ))}
        </div>

        {/* Room Chat */}
        <RoomChat roomID={room.id} userId={userId} />
      </div>

      {/* Left column (RTL: second in DOM = left side): VPN connection panel */}
      <div style={leftCol}>
        <div style={card}>
          <div style={sectionTitle}>اتصال VPN</div>
          <ConnectionStatus status={vpnStatus} pingMs={pingStats.last_ping} />

          {/* Stats grid */}
          <div style={statGrid}>
            <div style={statBox}>
              <div style={statLabel}>تأخیر</div>
              <div style={statValue(getPingColor(pingStats.last_ping))}>
                {pingStats.last_ping >= 0 ? `${pingStats.last_ping}` : "--"}
                <span style={{ fontSize: "0.7rem", color: "#8888aa", marginRight: "2px" }}>ms</span>
              </div>
            </div>
            <div style={statBox}>
              <div style={statLabel}>میانگین پینگ</div>
              <div style={statValue(getPingColor(pingStats.avg_ping))}>
                {pingStats.avg_ping >= 0 ? `${pingStats.avg_ping}` : "--"}
                <span style={{ fontSize: "0.7rem", color: "#8888aa", marginRight: "2px" }}>ms</span>
              </div>
            </div>
            <div style={statBox}>
              <div style={statLabel}>از دست رفتن بسته</div>
              <div
                style={statValue(
                  pingStats.packet_loss > 5
                    ? "#ff5252"
                    : pingStats.packet_loss > 2
                    ? "#ffab00"
                    : "#00e676"
                )}
              >
                {typeof pingStats.packet_loss === "number"
                  ? `${pingStats.packet_loss.toFixed(1)}`
                  : "--"}
                <span style={{ fontSize: "0.7rem", color: "#8888aa", marginRight: "2px" }}>%</span>
              </div>
            </div>
            <div style={statBox}>
              <div style={statLabel}>لرزش</div>
              <div
                style={statValue(
                  pingStats.jitter > 20
                    ? "#ff5252"
                    : pingStats.jitter > 10
                    ? "#ffab00"
                    : "#00e676"
                )}
              >
                {pingStats.jitter >= 0 ? `${pingStats.jitter}` : "--"}
                <span style={{ fontSize: "0.7rem", color: "#8888aa", marginRight: "2px" }}>ms</span>
              </div>
            </div>
          </div>

          {/* Quality bar */}
          <div style={{ marginBottom: "16px" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <span style={{ fontSize: "0.82rem", color: "#8888aa" }}>کیفیت</span>
              <span
                style={{
                  fontSize: "0.82rem",
                  fontWeight: 600,
                  color:
                    quality === "excellent"
                      ? "#00e676"
                      : quality === "good"
                      ? "#7c4dff"
                      : quality === "fair"
                      ? "#ffab00"
                      : quality === "poor"
                      ? "#ff5252"
                      : "#8888aa",
                }}
              >
                {QUALITY_LABELS[quality] || quality}
              </span>
            </div>
            <div
              style={{
                height: "4px",
                background: "#0d0d20",
                borderRadius: "2px",
                overflow: "hidden",
                marginTop: "6px",
              }}
            >
              <div style={qualityBar(quality)} />
            </div>
          </div>

          {/* Ping graph */}
          {pingHistory.length > 0 && <PingGraph pings={pingHistory} />}

          {/* Buttons */}
          <div style={{ ...btnRow, marginTop: "16px" }}>
            {vpnStatus === "disconnected" && (
              <button style={btnPrimary} onClick={handleConnect}>
                اتصال
              </button>
            )}
            {(vpnStatus === "connected" ||
              vpnStatus === "connecting" ||
              vpnStatus === "reconnecting") && (
              <button style={btnSecondary} onClick={handleDisconnect}>
                قطع اتصال
              </button>
            )}
            <button
              style={btnDanger}
              onClick={() => setShowLeaveConfirm(true)}
              disabled={leaving}
            >
              {leaving ? "در حال خروج..." : "خروج از اتاق"}
            </button>
          </div>

          {error && <div style={errorStyle}>{error}</div>}
        </div>

        {/* VPN Credentials */}
        {vpnCreds && (
          <div style={card}>
            <div style={sectionTitle}>اطلاعات اتصال VPN (دستی)</div>
            <div style={credCard}>
              {[
                { label: "میزبان", value: `${vpnCreds.vpn_host}:443`, key: "host" },
                { label: "هاب", value: vpnCreds.hub, key: "hub" },
                { label: "نام کاربری", value: vpnCreds.vpn_username, key: "user" },
                { label: "رمز عبور", value: vpnCreds.vpn_password, key: "pass" },
                { label: "زیرشبکه", value: vpnCreds.subnet, key: "subnet" },
              ].map((item) => (
                <div
                  key={item.key}
                  style={credRow}
                  onClick={() => handleCopy(item.value, item.key)}
                  title="برای کپی کلیک کنید"
                >
                  <span style={credLabel}>{item.label}</span>
                  <span style={{ display: "flex", alignItems: "center" }}>
                    <span style={credValue}>{item.value}</span>
                    <span style={copyHint}>
                      {copiedField === item.key ? "کپی شد!" : ""}
                    </span>
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Leave confirmation */}
      {showLeaveConfirm && (
        <div style={confirmOverlay} onClick={() => setShowLeaveConfirm(false)}>
          <div style={confirmBox} onClick={(e) => e.stopPropagation()}>
            <div
              style={{
                fontSize: "1.1rem",
                fontWeight: 600,
                color: "#e8e8f0",
                marginBottom: "12px",
              }}
            >
              خروج از اتاق؟
            </div>
            <p
              style={{
                color: "#8888aa",
                marginBottom: "24px",
                fontSize: "0.9rem",
                lineHeight: 1.5,
              }}
            >
              اتصال VPN قطع شده و از اتاق خارج خواهید شد.
            </p>
            <div style={{ display: "flex", gap: "10px", justifyContent: "center" }}>
              <button style={btnSecondary} onClick={() => setShowLeaveConfirm(false)}>
                انصراف
              </button>
              <button
                style={{ ...btnDanger, background: "#ff525218" }}
                onClick={handleLeave}
                disabled={leaving}
              >
                {leaving ? "در حال خروج..." : "خروج"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Extend Room modal */}
      {showExtend && (
        <div style={confirmOverlay} onClick={() => setShowExtend(false)}>
          <div style={confirmBox} onClick={(e) => e.stopPropagation()}>
            <div style={{ fontSize: "1.1rem", fontWeight: 600, color: "#e8e8f0", marginBottom: "16px" }}>
              {"🔶"} تمدید اتاق
            </div>
            <div style={{ marginBottom: "12px" }}>
              <select
                style={extendSelect}
                value={extendDuration}
                onChange={(e) => setExtendDuration(e.target.value)}
              >
                <option value="weekly">۷ روز</option>
                <option value="monthly">۱ ماه (۱۰٪ تخفیف)</option>
                <option value="quarterly">۳ ماه (۲۵٪ تخفیف)</option>
                <option value="yearly">سالانه (۴۰٪ تخفیف)</option>
              </select>
            </div>
            <div style={{
              padding: "12px",
              background: "#0d0d20",
              borderRadius: "8px",
              marginBottom: "16px",
              textAlign: "center",
            }}>
              <div style={{ color: SHARD_COLOR, fontSize: "1.2rem", fontWeight: 700, fontFamily: "monospace", direction: "ltr" }}>
                {"🔶"} {formatShards(
                  extendDuration === "yearly"
                    ? Math.round((room.max_players || 15) * 1000 * 365 * 0.6)
                    : extendDuration === "quarterly"
                    ? Math.round((room.max_players || 15) * 1000 * 90 * 0.75)
                    : extendDuration === "monthly"
                    ? Math.round((room.max_players || 15) * 1000 * 30 * 0.9)
                    : (room.max_players || 15) * 1000 * 7
                )}
              </div>
              <div style={{ color: "#8888aa", fontSize: "0.8rem", marginTop: "4px" }}>
                {room.max_players || 15} ظرفیت x {
                  extendDuration === "yearly" ? "365 روز" :
                  extendDuration === "quarterly" ? "90 روز" :
                  extendDuration === "monthly" ? "30 روز" : "7 روز"
                }
              </div>
            </div>
            <div style={{ display: "flex", gap: "10px", justifyContent: "center" }}>
              <button style={btnSecondary} onClick={() => setShowExtend(false)}>
                انصراف
              </button>
              <button
                style={{
                  ...btnShard,
                  opacity: extending ? 0.7 : 1,
                }}
                onClick={handleExtend}
                disabled={extending}
              >
                {extending ? "در حال تمدید..." : "تمدید"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Transfer Ownership modal */}
      {showTransfer && (
        <div style={confirmOverlay} onClick={() => setShowTransfer(false)}>
          <div style={confirmBox} onClick={(e) => e.stopPropagation()}>
            <div style={{ fontSize: "1.1rem", fontWeight: 600, color: "#e8e8f0", marginBottom: "16px" }}>
              انتقال مالکیت
            </div>
            <p style={{ color: "#8888aa", fontSize: "0.9rem", marginBottom: "16px" }}>
              عضوی را برای انتقال مالکیت اتاق انتخاب کنید:
            </p>
            <div style={{ maxHeight: "200px", overflowY: "auto", marginBottom: "16px" }}>
              {members.filter((m) => m.user_id !== userId).map((m) => (
                <div
                  key={m.user_id}
                  style={{
                    padding: "10px 14px",
                    borderRadius: "8px",
                    cursor: "pointer",
                    background: transferTarget === m.user_id ? `${SHARD_COLOR}18` : "transparent",
                    border: transferTarget === m.user_id ? `1px solid ${SHARD_COLOR}33` : "1px solid transparent",
                    marginBottom: "4px",
                    color: "#e8e8f0",
                    fontSize: "0.9rem",
                    transition: "all 0.15s ease",
                  }}
                  onClick={() => setTransferTarget(m.user_id)}
                >
                  {m.display_name}
                </div>
              ))}
              {members.filter((m) => m.user_id !== userId).length === 0 && (
                <div style={{ color: "#555577", fontSize: "0.85rem", textAlign: "center", padding: "12px" }}>
                  عضو دیگری برای انتقال وجود ندارد
                </div>
              )}
            </div>
            <div style={{ display: "flex", gap: "10px", justifyContent: "center" }}>
              <button style={btnSecondary} onClick={() => { setShowTransfer(false); setTransferTarget(null); }}>
                انصراف
              </button>
              <button
                style={{
                  ...btnDanger,
                  background: "#ff525218",
                  opacity: !transferTarget || transferring ? 0.6 : 1,
                }}
                onClick={handleTransferOwnership}
                disabled={!transferTarget || transferring}
              >
                {transferring ? "در حال انتقال..." : "انتقال"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
