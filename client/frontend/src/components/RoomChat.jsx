import React, { useState, useEffect, useRef, useCallback } from "react";
import { GetChatMessages, SendChatMessage } from "../api";

const chatContainer = {
  background: "#1a1a35",
  borderRadius: "12px",
  border: "1px solid #2a2a45",
  display: "flex",
  flexDirection: "column",
  height: "400px",
  direction: "rtl",
};

const chatHeader = {
  padding: "12px 16px",
  borderBottom: "1px solid #2a2a45",
  fontSize: "0.82rem",
  fontWeight: 600,
  color: "#8888aa",
  textTransform: "uppercase",
  letterSpacing: "0.5px",
  textAlign: "right",
};

const messageList = {
  flex: 1,
  overflowY: "auto",
  padding: "12px 16px",
  display: "flex",
  flexDirection: "column",
  gap: "8px",
};

const inputRow = {
  display: "flex",
  gap: "8px",
  padding: "12px 16px",
  borderTop: "1px solid #2a2a45",
  direction: "rtl",
};

const inputStyle = {
  flex: 1,
  padding: "10px 14px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#0d0d20",
  color: "#e8e8f0",
  fontSize: "0.9rem",
  outline: "none",
  direction: "rtl",
  textAlign: "right",
};

const sendBtn = {
  padding: "10px 20px",
  borderRadius: "10px",
  border: "none",
  background: "linear-gradient(135deg, #7c4dff, #6a3de8)",
  color: "#fff",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: "0.85rem",
  flexShrink: 0,
  transition: "all 0.2s ease",
};

const ownBubble = {
  alignSelf: "flex-end",
  background: "#7c4dff22",
  border: "1px solid #7c4dff33",
  borderRadius: "12px 12px 4px 12px",
  padding: "8px 14px",
  maxWidth: "75%",
  wordBreak: "break-word",
};

const otherBubble = {
  alignSelf: "flex-start",
  background: "#0d0d20",
  border: "1px solid #2a2a45",
  borderRadius: "12px 12px 12px 4px",
  padding: "8px 14px",
  maxWidth: "75%",
  wordBreak: "break-word",
};

const msgName = {
  fontSize: "0.72rem",
  fontWeight: 600,
  marginBottom: "4px",
};

const msgText = {
  fontSize: "0.88rem",
  color: "#e8e8f0",
  lineHeight: 1.5,
  textAlign: "right",
};

const msgTime = {
  fontSize: "0.68rem",
  color: "#555577",
  marginTop: "4px",
  textAlign: "left",
};

function formatTime(iso) {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } catch {
    return "";
  }
}

export default function RoomChat({ roomID, userId }) {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const lastMsgIDRef = useRef(0);
  const listRef = useRef(null);
  const pollRef = useRef(null);

  const fetchMessages = useCallback(async () => {
    if (!roomID) return;
    try {
      const msgs = await GetChatMessages(roomID, lastMsgIDRef.current);
      if (msgs && msgs.length > 0) {
        setMessages((prev) => {
          const combined = [...prev, ...msgs];
          // Deduplicate by id
          const seen = new Set();
          const unique = combined.filter((m) => {
            if (seen.has(m.id)) return false;
            seen.add(m.id);
            return true;
          });
          return unique;
        });
        const maxID = Math.max(...msgs.map((m) => m.id || 0));
        if (maxID > lastMsgIDRef.current) {
          lastMsgIDRef.current = maxID;
        }
      }
    } catch {
      // ignore
    }
  }, [roomID]);

  useEffect(() => {
    fetchMessages();
    pollRef.current = setInterval(fetchMessages, 3000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [fetchMessages]);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = async () => {
    const content = input.trim();
    if (!content || !roomID) return;
    if (content.length > 500) return;
    setSending(true);
    try {
      await SendChatMessage(roomID, content);
      setInput("");
      // Fetch immediately to show the new message
      await fetchMessages();
    } catch {
      // ignore
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div style={chatContainer}>
      <div style={chatHeader}>{"\u0686\u062A \u0627\u062A\u0627\u0642"}</div>
      <div style={messageList} ref={listRef}>
        {messages.length === 0 && (
          <div style={{ color: "#555577", fontSize: "0.85rem", textAlign: "center", padding: "24px 0" }}>
            {"\u0647\u0646\u0648\u0632 \u067E\u06CC\u0627\u0645\u06CC \u0627\u0631\u0633\u0627\u0644 \u0646\u0634\u062F\u0647"}
          </div>
        )}
        {messages.map((m) => {
          const isOwn = m.user_id === userId;
          return (
            <div key={m.id} style={isOwn ? ownBubble : otherBubble}>
              {!isOwn && (
                <div style={{ ...msgName, color: "#7c4dff" }}>{m.display_name}</div>
              )}
              <div style={msgText}>{m.content}</div>
              <div style={msgTime}>{formatTime(m.created_at)}</div>
            </div>
          );
        })}
      </div>
      <div style={inputRow}>
        <input
          style={inputStyle}
          placeholder={"\u067E\u06CC\u0627\u0645 \u062E\u0648\u062F \u0631\u0627 \u0628\u0646\u0648\u06CC\u0633\u06CC\u062F..."}
          value={input}
          onChange={(e) => {
            if (e.target.value.length <= 500) setInput(e.target.value);
          }}
          onKeyDown={handleKeyDown}
          maxLength={500}
          disabled={sending}
        />
        <button
          style={{ ...sendBtn, opacity: sending ? 0.7 : 1 }}
          onClick={handleSend}
          disabled={sending || !input.trim()}
        >
          {"\u0627\u0631\u0633\u0627\u0644"}
        </button>
      </div>
    </div>
  );
}
