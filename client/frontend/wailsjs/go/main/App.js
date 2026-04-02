// @ts-check
// This file is a placeholder for Wails auto-generated bindings.
// During `wails dev` or `wails build`, Wails generates this file automatically
// based on the Go methods bound in main.go. These stubs allow the frontend
// code to import without errors before the first build.

export function Register(phone, password, displayName, referralCode) {
  return window["go"]["main"]["App"]["Register"](phone, password, displayName, referralCode);
}

export function GetDeviceFingerprint() {
  return window["go"]["main"]["App"]["GetDeviceFingerprint"]();
}

export function Login(phone, password) {
  return window["go"]["main"]["App"]["Login"](phone, password);
}

export function ListRooms(query, isPrivate, hasSlots, page) {
  return window["go"]["main"]["App"]["ListRooms"](query, isPrivate, hasSlots, page);
}

export function GetRoom(roomID) {
  return window["go"]["main"]["App"]["GetRoom"](roomID);
}

export function JoinRoom(roomID, password) {
  return window["go"]["main"]["App"]["JoinRoom"](roomID, password);
}

export function LeaveRoom(roomID) {
  return window["go"]["main"]["App"]["LeaveRoom"](roomID);
}

export function GetMembers(roomID) {
  return window["go"]["main"]["App"]["GetMembers"](roomID);
}

export function GetBuyInfo() {
  return window["go"]["main"]["App"]["GetBuyInfo"]();
}

export function ConnectVPN(host, hub, username, password, subnet) {
  return window["go"]["main"]["App"]["ConnectVPN"](host, hub, username, password, subnet);
}

export function DisconnectVPN() {
  return window["go"]["main"]["App"]["DisconnectVPN"]();
}

export function StopVPN() {
  return window["go"]["main"]["App"]["StopVPN"]();
}

export function GetVPNStatus() {
  return window["go"]["main"]["App"]["GetVPNStatus"]();
}

export function PingServer(host) {
  return window["go"]["main"]["App"]["PingServer"](host);
}

export function GetPingStats() {
  return window["go"]["main"]["App"]["GetPingStats"]();
}

export function GetConnectionQuality() {
  return window["go"]["main"]["App"]["GetConnectionQuality"]();
}

export function CheckVPNReady() {
  return window["go"]["main"]["App"]["CheckVPNReady"]();
}

export function IsSplitTunnelActive() {
  return window["go"]["main"]["App"]["IsSplitTunnelActive"]();
}

export function SetServerURL(url) {
  return window["go"]["main"]["App"]["SetServerURL"](url);
}

export function GetServerURL() {
  return window["go"]["main"]["App"]["GetServerURL"]();
}

export function CheckSoftEtherInstalled() {
  return window["go"]["main"]["App"]["CheckSoftEtherInstalled"]();
}

export function GetSoftEtherVersion() {
  return window["go"]["main"]["App"]["GetSoftEtherVersion"]();
}

export function EnsureSoftEtherRunning() {
  return window["go"]["main"]["App"]["EnsureSoftEtherRunning"]();
}

// --- Shard-based business model methods ---

export function GetPricing() {
  return window["go"]["main"]["App"]["GetPricing"]();
}

export function GetShopInfo() {
  return window["go"]["main"]["App"]["GetShopInfo"]();
}

export function PurchaseRoom(name, gameTag, slots, duration, days, isPrivate, password) {
  return window["go"]["main"]["App"]["PurchaseRoom"](name, gameTag, slots, duration, days, isPrivate, password);
}

export function ExtendRoom(roomID, duration, days) {
  return window["go"]["main"]["App"]["ExtendRoom"](roomID, duration, days);
}

export function SetRoomRole(roomID, userID, role) {
  return window["go"]["main"]["App"]["SetRoomRole"](roomID, userID, role);
}

export function TransferRoom(roomID, userID) {
  return window["go"]["main"]["App"]["TransferRoom"](roomID, userID);
}

export function GetMyStats() {
  return window["go"]["main"]["App"]["GetMyStats"]();
}

export function GetMe() {
  return window["go"]["main"]["App"]["GetMe"]();
}

// --- Auto-update ---

export function CheckForUpdate() {
  return window["go"]["main"]["App"]["CheckForUpdate"]();
}

export function GetCurrentVersion() {
  return window["go"]["main"]["App"]["GetCurrentVersion"]();
}

// --- Local VPN IP ---

export function GetLocalVPNIP() {
  return window["go"]["main"]["App"]["GetLocalVPNIP"]();
}

// --- Room chat ---

export function SendChatMessage(roomID, content) {
  return window["go"]["main"]["App"]["SendChatMessage"](roomID, content);
}

export function GetChatMessages(roomID, afterID) {
  return window["go"]["main"]["App"]["GetChatMessages"](roomID, afterID);
}

// --- Invite system ---

export function CreateInvite(roomID, maxUses, expiresHours) {
  return window["go"]["main"]["App"]["CreateInvite"](roomID, maxUses, expiresHours);
}

export function JoinByInvite(token) {
  return window["go"]["main"]["App"]["JoinByInvite"](token);
}

// --- Password change ---

export function ChangePassword(oldPassword, newPassword) {
  return window["go"]["main"]["App"]["ChangePassword"](oldPassword, newPassword);
}

// --- Promo & referrals ---

export function RedeemPromo(code) {
  return window["go"]["main"]["App"]["RedeemPromo"](code);
}

export function GetReferralInfo() {
  return window["go"]["main"]["App"]["GetReferralInfo"]();
}

// --- Balance ---

export function RefreshBalance() {
  return window["go"]["main"]["App"]["RefreshBalance"]();
}
