import React, { useState } from "react";
import LocalIP from "../components/LocalIP";

const pageStyle = {
  maxWidth: "700px",
  margin: "0 auto",
  display: "flex",
  flexDirection: "column",
  gap: "16px",
  animation: "fadeIn 0.3s ease-out",
  direction: "rtl",
};

const card = {
  background: "#1a1a35",
  borderRadius: "12px",
  padding: "24px",
  border: "1px solid #2a2a45",
};

const pageTitle = {
  fontSize: "1.3rem",
  fontWeight: 700,
  color: "#e8e8f0",
  textAlign: "right",
  marginBottom: "4px",
};

const pageSubtitle = {
  fontSize: "0.88rem",
  color: "#8888aa",
  textAlign: "right",
};

const ipBanner = {
  background: "#0d0d20",
  borderRadius: "12px",
  padding: "20px 24px",
  border: "1px solid #2a2a45",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
};

const accordionHeader = (isOpen) => ({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "16px 20px",
  background: isOpen ? "#1e1e3a" : "#1a1a35",
  borderRadius: isOpen ? "12px 12px 0 0" : "12px",
  border: `1px solid ${isOpen ? "#3a3a55" : "#2a2a45"}`,
  cursor: "pointer",
  transition: "all 0.2s ease",
  direction: "rtl",
});

const accordionBody = {
  background: "#141428",
  borderRadius: "0 0 12px 12px",
  border: "1px solid #2a2a45",
  borderTop: "none",
  padding: "20px",
  direction: "rtl",
};

const gameTitle = {
  fontSize: "1rem",
  fontWeight: 700,
  color: "#e8e8f0",
  display: "flex",
  alignItems: "center",
  gap: "10px",
};

const chevron = (isOpen) => ({
  color: "#8888aa",
  fontSize: "0.8rem",
  transform: isOpen ? "rotate(90deg)" : "rotate(0deg)",
  transition: "transform 0.2s ease",
});

const codeBlock = {
  background: "#0d0d20",
  borderRadius: "8px",
  padding: "14px 16px",
  fontFamily: "monospace",
  fontSize: "0.85rem",
  color: "#e8e8f0",
  lineHeight: 1.8,
  border: "1px solid #1a1a30",
  overflowX: "auto",
  direction: "ltr",
  textAlign: "left",
  whiteSpace: "pre-wrap",
  marginTop: "10px",
  marginBottom: "10px",
};

const stepText = {
  fontSize: "0.9rem",
  color: "#e8e8f0",
  lineHeight: 1.8,
  textAlign: "right",
};

const noteBox = {
  background: "#ffab0008",
  borderRadius: "8px",
  padding: "12px 16px",
  border: "1px solid #ffab0022",
  fontSize: "0.85rem",
  color: "#ffab00",
  lineHeight: 1.6,
  marginTop: "10px",
  textAlign: "right",
};

const hostIPNote = {
  background: "#7c4dff08",
  borderRadius: "8px",
  padding: "10px 14px",
  border: "1px solid #7c4dff22",
  fontSize: "0.82rem",
  color: "#7c4dff",
  lineHeight: 1.5,
  marginTop: "8px",
  textAlign: "right",
};

const GAMES = [
  {
    id: "dota2",
    icon: "\u2694",
    title: "\u062F\u0648\u062A\u0627 \u06F2 - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={stepText}>
          {"\u06F1. \u0648\u0627\u0631\u062F \u0628\u0627\u0632\u06CC \u0634\u0648\u06CC\u062F"}
          <br />
          {"\u06F2. \u062F\u0631 \u06A9\u0646\u0633\u0648\u0644 \u062A\u0627\u06CC\u067E \u06A9\u0646\u06CC\u062F: (\u062F\u06A9\u0645\u0647 ~ \u0631\u0627 \u0628\u0632\u0646\u06CC\u062F)"}
        </div>
        <div style={codeBlock}>connect &lt;IP&gt;</div>
        <div style={hostIPNote}>
          {"\u0622\u06CC\u200C\u067E\u06CC \u0628\u0627\u0632\u06CC\u06A9\u0646 \u0645\u06CC\u0632\u0628\u0627\u0646 \u0631\u0627 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F"}
        </div>
        <div style={stepText}>
          {"\u06CC\u0627 \u0627\u0632 \u0645\u0646\u0648\u06CC Play \u2190 Custom Lobby \u2190 Create Lobby \u0627\u0633\u062A\u0641\u0627\u062F\u0647 \u06A9\u0646\u06CC\u062F"}
          <br /><br />
          {"\u06F3. \u062A\u0646\u0638\u06CC\u0645\u0627\u062A \u0644\u0627\u0628\u06CC:"}
          <br />
          {"   - Server Location: Local"}
          <br />
          {"   - Game Mode: \u062F\u0644\u062E\u0648\u0627\u0647"}
        </div>
        <div style={{ ...stepText, marginTop: "12px" }}>
          {"\u062F\u0633\u062A\u0648\u0631\u0627\u062A \u0645\u0641\u06CC\u062F \u06A9\u0646\u0633\u0648\u0644:"}
        </div>
        <div style={codeBlock}>
{`status          - \u0646\u0645\u0627\u06CC\u0634 \u0648\u0636\u0639\u06CC\u062A \u0633\u0631\u0648\u0631
ping            - \u0646\u0645\u0627\u06CC\u0634 \u067E\u06CC\u0646\u06AF
disconnect      - \u0642\u0637\u0639 \u0627\u062A\u0635\u0627\u0644
map dota        - \u0634\u0631\u0648\u0639 \u0628\u0627\u0632\u06CC \u062C\u062F\u06CC\u062F`}
        </div>
      </>
    ),
  },
  {
    id: "cs2",
    icon: "\u{1F3AF}",
    title: "CS2 - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={stepText}>
          {"\u06F1. \u062F\u0631 \u06A9\u0646\u0633\u0648\u0644 \u062A\u0627\u06CC\u067E \u06A9\u0646\u06CC\u062F:"}
        </div>
        <div style={codeBlock}>connect &lt;IP&gt;</div>
        <div style={hostIPNote}>
          {"\u0622\u06CC\u200C\u067E\u06CC \u0628\u0627\u0632\u06CC\u06A9\u0646 \u0645\u06CC\u0632\u0628\u0627\u0646 \u0631\u0627 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F"}
        </div>
        <div style={stepText}>
          <br />
          {"\u06F2. \u06CC\u0627 \u0633\u0631\u0648\u0631 \u0627\u062E\u062A\u0635\u0627\u0635\u06CC \u0628\u0633\u0627\u0632\u06CC\u062F:"}
          <br />
          {"   - Play \u2190 Practice \u2190 Competitive"}
        </div>
        <div style={{ ...stepText, marginTop: "12px" }}>
          {"\u062F\u0633\u062A\u0648\u0631\u0627\u062A \u06A9\u0646\u0633\u0648\u0644:"}
        </div>
        <div style={codeBlock}>
{`status                    - \u0648\u0636\u0639\u06CC\u062A \u0633\u0631\u0648\u0631
sv_lan 1                  - \u0641\u0639\u0627\u0644 \u06A9\u0631\u062F\u0646 \u062D\u0627\u0644\u062A \u0644\u0646
changelevel de_dust2      - \u062A\u063A\u06CC\u06CC\u0631 \u0645\u067E
mp_autoteambalance 0      - \u063A\u06CC\u0631\u0641\u0639\u0627\u0644 \u06A9\u0631\u062F\u0646 \u0628\u0627\u0644\u0627\u0646\u0633 \u062A\u06CC\u0645
mp_warmuptime 0           - \u0631\u062F \u06A9\u0631\u062F\u0646 \u0648\u0627\u0631\u0645\u200C\u0622\u067E`}
        </div>
      </>
    ),
  },
  {
    id: "wc3",
    icon: "\u{1F9D9}",
    title: "\u0648\u0627\u0631\u06A9\u0631\u0627\u0641\u062A \u06F3 - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={stepText}>
          {"\u06F1. \u0648\u0627\u0631\u062F \u0628\u0627\u0632\u06CC \u0634\u0648\u06CC\u062F"}
          <br />
          {"\u06F2. Local Area Network \u0631\u0627 \u0627\u0646\u062A\u062E\u0627\u0628 \u06A9\u0646\u06CC\u062F"}
          <br />
          {"\u06F3. \u0628\u0627\u0632\u06CC \u0631\u0627 \u0627\u06CC\u062C\u0627\u062F \u06A9\u0646\u06CC\u062F \u06CC\u0627 \u0628\u0627\u0632\u06CC \u0645\u0648\u062C\u0648\u062F \u0631\u0627 \u0628\u0628\u06CC\u0646\u06CC\u062F"}
          <br />
          {"\u06F4. \u0628\u0627\u0632\u06CC\u06A9\u0646\u0627\u0646 \u062F\u06CC\u06AF\u0631 \u0628\u0627\u06CC\u062F \u062F\u0631 \u0647\u0645\u0627\u0646 \u0644\u0646 \u0628\u0627\u0634\u0646\u062F"}
        </div>
        <div style={noteBox}>
          {"\u0646\u06A9\u062A\u0647: \u0627\u06AF\u0631 \u0628\u0627\u0632\u06CC \u0631\u0627 \u0646\u0645\u06CC\u200C\u0628\u06CC\u0646\u06CC\u062F\u060C \u0641\u0627\u06CC\u0631\u0648\u0627\u0644 \u0648\u06CC\u0646\u062F\u0648\u0632 \u0631\u0627 \u0686\u06A9 \u06A9\u0646\u06CC\u062F"}
        </div>
      </>
    ),
  },
  {
    id: "aoe2",
    icon: "\u{1F3F0}",
    title: "\u0639\u0635\u0631 \u0627\u0645\u067E\u0631\u0627\u0637\u0648\u0631\u06CC \u06F2 - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={stepText}>
          {"\u06F1. Multiplayer \u2190 LAN"}
          <br />
          {"\u06F2. Create Game \u06CC\u0627 Browse Games"}
          <br />
          {"\u06F3. \u0645\u0637\u0645\u0626\u0646 \u0634\u0648\u06CC\u062F \u0647\u0645\u0647 \u062F\u0631 \u06CC\u06A9 \u0632\u06CC\u0631\u0634\u0628\u06A9\u0647 \u0647\u0633\u062A\u06CC\u062F"}
        </div>
      </>
    ),
  },
  {
    id: "valorant",
    icon: "\u{1F52B}",
    title: "\u0648\u0644\u0648\u0631\u0646\u062A - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={noteBox}>
          {"\u0646\u06A9\u062A\u0647: \u0648\u0644\u0648\u0631\u0646\u062A \u062D\u0627\u0644\u062A \u0644\u0646 \u0646\u062F\u0627\u0631\u062F \u0648\u0644\u06CC \u0645\u06CC\u200C\u062A\u0648\u0627\u0646\u06CC\u062F Custom Game \u0628\u0633\u0627\u0632\u06CC\u062F"}
        </div>
        <div style={stepText}>
          {"\u06F1. Play \u2190 Custom Game \u2190 Create"}
          <br />
          {"\u06F2. \u062F\u0648\u0633\u062A\u0627\u0646 \u0631\u0627 \u062F\u0639\u0648\u062A \u06A9\u0646\u06CC\u062F"}
        </div>
      </>
    ),
  },
  {
    id: "minecraft",
    icon: "\u26CF",
    title: "\u0645\u0627\u06CC\u0646\u200C\u06A9\u0631\u0641\u062A - \u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC \u0644\u0646",
    content: (
      <>
        <div style={stepText}>
          {"\u06F1. Open to LAN \u0631\u0627 \u062F\u0631 \u0645\u0646\u0648\u06CC Pause \u0628\u0632\u0646\u06CC\u062F"}
          <br />
          {"\u06F2. \u067E\u0648\u0631\u062A \u0646\u0645\u0627\u06CC\u0634 \u062F\u0627\u062F\u0647 \u0634\u062F\u0647 \u0631\u0627 \u0628\u0647 \u0628\u0642\u06CC\u0647 \u0628\u062F\u0647\u06CC\u062F"}
          <br />
          {"\u06F3. \u0628\u0627\u0632\u06CC\u06A9\u0646\u0627\u0646 \u062F\u06CC\u06AF\u0631: Multiplayer \u2190 Direct Connect \u2190 <IP>:<PORT>"}
        </div>
        <div style={hostIPNote}>
          {"\u0622\u06CC\u200C\u067E\u06CC \u0628\u0627\u0632\u06CC\u06A9\u0646 \u0645\u06CC\u0632\u0628\u0627\u0646 \u0631\u0627 \u0648\u0627\u0631\u062F \u06A9\u0646\u06CC\u062F"}
        </div>
      </>
    ),
  },
];

export default function GameGuidesPage() {
  const [openSections, setOpenSections] = useState({});

  const toggleSection = (id) => {
    setOpenSections((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  return (
    <div style={pageStyle}>
      {/* Page title */}
      <div style={card}>
        <div style={pageTitle}>{"\u0631\u0627\u0647\u0646\u0645\u0627\u06CC \u0628\u0627\u0632\u06CC\u200C\u0647\u0627"}</div>
        <div style={pageSubtitle}>{"\u062F\u0633\u062A\u0648\u0631\u0627\u0644\u0639\u0645\u0644 \u0627\u062A\u0635\u0627\u0644 \u0644\u0646 \u0628\u0631\u0627\u06CC \u0647\u0631 \u0628\u0627\u0632\u06CC"}</div>
      </div>

      {/* LAN IP Banner */}
      <div style={ipBanner}>
        <LocalIP />
      </div>

      {/* Accordion sections per game */}
      {GAMES.map((game) => {
        const isOpen = !!openSections[game.id];
        return (
          <div key={game.id}>
            <div
              style={accordionHeader(isOpen)}
              onClick={() => toggleSection(game.id)}
              onMouseEnter={(e) => {
                if (!isOpen) e.currentTarget.style.borderColor = "#3a3a55";
              }}
              onMouseLeave={(e) => {
                if (!isOpen) e.currentTarget.style.borderColor = "#2a2a45";
              }}
            >
              <div style={gameTitle}>
                <span>{game.icon}</span>
                <span>{game.title}</span>
              </div>
              <span style={chevron(isOpen)}>{"\u25B6"}</span>
            </div>
            {isOpen && <div style={accordionBody}>{game.content}</div>}
          </div>
        );
      })}
    </div>
  );
}
