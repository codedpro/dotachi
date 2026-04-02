import React from "react";

const wrapperStyle = {
  position: "relative",
  direction: "rtl",
};

const inputStyle = {
  width: "100%",
  padding: "12px 42px 12px 18px",
  borderRadius: "10px",
  border: "1px solid #2a2a45",
  background: "#141428",
  color: "#e8e8f0",
  fontSize: "0.95rem",
  outline: "none",
  transition: "border-color 0.2s ease, box-shadow 0.2s ease",
  direction: "rtl",
  textAlign: "right",
  fontFamily: "'Vazirmatn', sans-serif",
};

const iconStyle = {
  position: "absolute",
  right: "14px",
  top: "50%",
  transform: "translateY(-50%)",
  color: "#8888aa",
  fontSize: "0.95rem",
  pointerEvents: "none",
  userSelect: "none",
};

export default function SearchBar({ value, onChange, placeholder }) {
  return (
    <div style={wrapperStyle}>
      <span style={iconStyle}>&#9906;</span>
      <input
        style={inputStyle}
        type="text"
        placeholder={placeholder || "جستجوی اتاق‌ها..."}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={(e) => {
          e.target.style.borderColor = "#7c4dff66";
          e.target.style.boxShadow = "0 0 0 2px #7c4dff22";
        }}
        onBlur={(e) => {
          e.target.style.borderColor = "#2a2a45";
          e.target.style.boxShadow = "none";
        }}
      />
    </div>
  );
}
