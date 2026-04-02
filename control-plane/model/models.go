package model

type User struct {
	ID             int64   `json:"id"`
	Phone          string  `json:"phone,omitempty"`
	DisplayName    string  `json:"display_name"`
	IsAdmin        bool    `json:"is_admin"`
	CreatedAt      string  `json:"created_at"`
	ShardBalance   int     `json:"shard_balance"`
	TotalPlayHours float64 `json:"total_play_hours"`
	TotalSessions  int     `json:"total_sessions"`
}

type Node struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	APIPort       int     `json:"api_port"`
	APISecret     string  `json:"-"`
	IsActive      bool    `json:"is_active"`
	MaxRooms      int     `json:"max_rooms"`
	CreatedAt     string  `json:"created_at"`
	RoomCount     int     `json:"room_count,omitempty"`
	LastHeartbeat *string `json:"last_heartbeat,omitempty"`
	HubCount      int     `json:"hub_count,omitempty"`
	SessionCount  int     `json:"session_count,omitempty"`
	CPUUsage      float64 `json:"cpu_usage,omitempty"`
	MemoryMB      int     `json:"memory_mb,omitempty"`
}

type Room struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	HubName          string  `json:"hub_name"`
	NodeID           int64   `json:"node_id"`
	NodeName         string  `json:"node_name,omitempty"`
	OwnerID          *int64  `json:"owner_id"`
	OwnerDisplayName string  `json:"owner_display_name,omitempty"`
	IsPrivate        bool    `json:"is_private"`
	MaxPlayers       int     `json:"max_players"`
	CurrentPlayers   int     `json:"current_players"`
	Subnet           string  `json:"subnet"`
	IsActive         bool    `json:"is_active"`
	GameTag          string  `json:"game_tag"`
	Description      string  `json:"description"`
	ExpiresAt        *string `json:"expires_at,omitempty"`
	IsShared         bool    `json:"is_shared"`
	HourlyCost       int     `json:"hourly_cost"`
	CreatedAt        string  `json:"created_at"`
}

type RoomMember struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	JoinedAt    string `json:"joined_at"`
}

type JoinResponse struct {
	VPNHost     string `json:"vpn_host"`
	Hub         string `json:"hub"`
	VPNUsername string `json:"vpn_username"`
	VPNPassword string `json:"vpn_password"`
	Subnet      string `json:"subnet"`
}

type Favorite struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	RoomID    int64  `json:"room_id"`
	CreatedAt string `json:"created_at"`
}

type PlayerStats struct {
	TotalPlayHours float64 `json:"total_play_hours"`
	TotalSessions  int     `json:"total_sessions"`
	ShardBalance   int     `json:"shard_balance"`
	RoomsOwned     int     `json:"rooms_owned"`
	FavoriteCount  int     `json:"favorite_count"`
	MemberSince    string  `json:"member_since"`
}

type RoomRole struct {
	RoomID int    `json:"room_id"`
	UserID int    `json:"user_id"`
	Role   string `json:"role"` // owner, admin, member
}

type ShardTransaction struct {
	ID           int    `json:"id"`
	Amount       int    `json:"amount"`
	BalanceAfter int    `json:"balance_after"`
	TxType       string `json:"tx_type"`
	Description  string `json:"description"`
	CreatedAt    string `json:"created_at"`
}

type RoomPricing struct {
	Slots        int `json:"slots"`
	DailyPrice   int `json:"daily_price"`
	MonthlyPrice int `json:"monthly_price"`
	YearlyPrice  int `json:"yearly_price"`
}

type ShopInfo struct {
	ShardsPerToman  int           `json:"shards_per_toman"`
	ContactTelegram string        `json:"contact_telegram"`
	ContactBale     string        `json:"contact_bale"`
	Pricing         []RoomPricing `json:"pricing"`
}
