import React, { useState, useEffect, useRef, useCallback } from "react";
import { GetLocalVPNIP } from "../api";

export default function LocalIP() {
  const [ip, setIp] = useState("");
  const [copied, setCopied] = useState(false);
  const timerRef = useRef(null);
  const copyTimerRef = useRef(null);

  const fetchIP = useCallback(async () => {
    try {
      const result = await GetLocalVPNIP();
      if (result) setIp(result);
      else setIp("");
    } catch {
      setIp("");
    }
  }, []);

  useEffect(() => {
    fetchIP();
    timerRef.current = setInterval(fetchIP, 5000);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
    };
  }, [fetchIP]);

  const handleCopy = () => {
    if (!ip) return;
    try {
      navigator.clipboard.writeText(ip);
      setCopied(true);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      copyTimerRef.current = setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <div style={{ display: "flex", alignItems: "center", gap: "8px", direction: "rtl" }}>
      <span style={{ color: "#8888aa", fontSize: "0.88rem" }}>{"\u0622\u06CC\u200C\u067E\u06CC \u0644\u0646:"}</span>
      <span
        onClick={handleCopy}
        style={{
          fontFamily: "monospace",
          color: copied ? "#00e676" : "#e8e8f0",
          cursor: ip ? "pointer" : "default",
          transition: "color 0.3s",
          fontSize: "0.95rem",
          fontWeight: 600,
          userSelect: "all",
        }}
      >
        {ip || "---"}
      </span>
      {copied && (
        <span
          style={{
            color: "#00e676",
            fontSize: "0.75rem",
            transition: "opacity 0.5s",
          }}
        >
          {"\u06A9\u067E\u06CC \u0634\u062F"}
        </span>
      )}
    </div>
  );
}
