package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dotachi-cli",
		Short: "Dotachi platform admin CLI",
	}

	rootCmd.AddCommand(
		loginCmd(),
		nodesCmd(),
		roomsCmd(),
		usersCmd(),
		shardsCmd(),
		pricingCmd(),
		statusCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ──────────────────────────────────────────────
// login
// ──────────────────────────────────────────────

func loginCmd() *cobra.Command {
	var phone, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and store JWT token",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post("/auth/login", map[string]string{
				"phone":    phone,
				"password": password,
			})
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			token, ok := resp["access_token"].(string)
			if !ok {
				// Fallback to "token" key for compatibility.
				token, ok = resp["token"].(string)
			}
			if !ok {
				return fmt.Errorf("no token in response")
			}

			if err := SaveToken(token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}

			fmt.Println("Login successful. Token saved to ~/.dotachi/token")
			return nil
		},
	}

	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&password, "password", "", "Password")
	cmd.MarkFlagRequired("phone")
	cmd.MarkFlagRequired("password")
	return cmd
}

// ──────────────────────────────────────────────
// nodes
// ──────────────────────────────────────────────

func nodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Manage game nodes",
	}

	cmd.AddCommand(nodesListCmd(), nodesAddCmd(), nodesPingCmd())
	return cmd
}

func nodesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			rows, err := c.GetList("/nodes")
			if err != nil {
				return err
			}

			PrintTable([]Column{
				{"ID", "id"},
				{"Name", "name"},
				{"Host", "host"},
				{"Port", "port"},
				{"Rooms", "rooms"},
				{"Status", "status"},
			}, rows)
			return nil
		},
	}
}

func nodesAddCmd() *cobra.Command {
	var name, host, secret string
	var port int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register a new node",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post("/nodes", map[string]interface{}{
				"name":   name,
				"host":   host,
				"port":   port,
				"secret": secret,
			})
			if err != nil {
				return err
			}

			fmt.Println("Node added.")
			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Node name")
	cmd.Flags().StringVar(&host, "host", "", "Node IP or hostname")
	cmd.Flags().IntVar(&port, "port", 7443, "Node port")
	cmd.Flags().StringVar(&secret, "secret", "", "Shared secret")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("host")
	cmd.MarkFlagRequired("secret")
	return cmd
}

func nodesPingCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping a node to check health",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post(fmt.Sprintf("/nodes/%s/ping", id), nil)
			if err != nil {
				return err
			}

			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Node ID")
	cmd.MarkFlagRequired("id")
	return cmd
}

// ──────────────────────────────────────────────
// rooms
// ──────────────────────────────────────────────

func roomsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rooms",
		Short: "Manage game rooms",
	}

	cmd.AddCommand(roomsListCmd(), roomsCreateCmd(), roomsAssignCmd(), roomsDeleteCmd())
	return cmd
}

func roomsListCmd() *cobra.Command {
	var node, search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rooms with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()

			path := "/rooms"
			params := url.Values{}
			if node != "" {
				params.Set("node", node)
			}
			if search != "" {
				params.Set("search", search)
			}
			if len(params) > 0 {
				path += "?" + params.Encode()
			}

			rows, err := c.GetList(path)
			if err != nil {
				return err
			}

			PrintTable([]Column{
				{"ID", "id"},
				{"Name", "name"},
				{"Node", "node_id"},
				{"Players", "players"},
				{"Max", "max_players"},
				{"Private", "private"},
				{"Status", "status"},
				{"Expires", "expires_at"},
			}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&node, "node", "", "Filter by node ID")
	cmd.Flags().StringVar(&search, "search", "", "Search by room name")
	return cmd
}

func roomsCreateCmd() *cobra.Command {
	var name string
	var nodeID, maxPlayers int
	var private, shared bool
	var password, expires, game string
	var hourlyCost int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new room",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()

			body := map[string]interface{}{
				"name":        name,
				"node_id":     nodeID,
				"max_players": maxPlayers,
				"private":     private,
				"shared":      shared,
			}
			if password != "" {
				body["password"] = password
			}
			if expires != "" {
				body["expires"] = expires
			}
			if hourlyCost > 0 {
				body["hourly_cost"] = hourlyCost
			}
			if game != "" {
				body["game"] = game
			}

			resp, err := c.Post("/admin/rooms", body)
			if err != nil {
				return err
			}

			fmt.Println("Room created.")
			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Room name")
	cmd.Flags().IntVar(&nodeID, "node", 0, "Node ID to host the room")
	cmd.Flags().IntVar(&maxPlayers, "max-players", 10, "Maximum players")
	cmd.Flags().BoolVar(&private, "private", false, "Make room private")
	cmd.Flags().StringVar(&password, "password", "", "Room password (implies private)")
	cmd.Flags().StringVar(&expires, "expires", "", "Room duration (e.g. 7d, 30d, 365d)")
	cmd.Flags().BoolVar(&shared, "shared", false, "Allow shared access to the room")
	cmd.Flags().IntVar(&hourlyCost, "hourly-cost", 0, "Hourly shard cost for the room")
	cmd.Flags().StringVar(&game, "game", "", "Game type for the room")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("node")
	return cmd
}

func roomsAssignCmd() *cobra.Command {
	var roomID, userID int

	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign an owner to a room",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post(fmt.Sprintf("/admin/rooms/%d/assign-owner", roomID), map[string]interface{}{
				"user_id": userID,
			})
			if err != nil {
				return err
			}

			fmt.Println("Owner assigned.")
			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&roomID, "room", 0, "Room ID")
	cmd.Flags().IntVar(&userID, "user", 0, "User ID")
	cmd.MarkFlagRequired("room")
	cmd.MarkFlagRequired("user")
	return cmd
}

func roomsDeleteCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a room",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			_, err := c.Delete(fmt.Sprintf("/rooms/%s", id))
			if err != nil {
				return err
			}

			fmt.Println("Room deleted.")
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Room ID")
	cmd.MarkFlagRequired("id")
	return cmd
}

// ──────────────────────────────────────────────
// users
// ──────────────────────────────────────────────

func usersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
	}

	cmd.AddCommand(usersListCmd())
	return cmd
}

func usersListCmd() *cobra.Command {
	var search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users with optional search",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()

			path := "/admin/users"
			if search != "" {
				path += "?" + url.Values{"search": {search}}.Encode()
			}

			rows, err := c.GetList(path)
			if err != nil {
				return err
			}

			PrintTable([]Column{
				{"ID", "id"},
				{"Name", "name"},
				{"Phone", "phone"},
				{"Role", "role"},
				{"Shards", "shard_balance"},
				{"Created", "created_at"},
			}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Search by phone or name")
	return cmd
}

// ──────────────────────────────────────────────
// shards
// ──────────────────────────────────────────────

func shardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shards",
		Short: "Manage user shard balances",
	}

	cmd.AddCommand(shardsAddCmd(), shardsRemoveCmd())
	return cmd
}

func shardsAddCmd() *cobra.Command {
	var userID, amount int
	var description string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add shards to a user's balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post(fmt.Sprintf("/admin/users/%d/add-shards", userID), map[string]interface{}{
				"amount":      amount,
				"description": description,
			})
			if err != nil {
				return err
			}

			fmt.Println("Shards added.")
			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&userID, "user", 0, "User ID")
	cmd.Flags().IntVar(&amount, "amount", 0, "Number of shards to add")
	cmd.Flags().StringVar(&description, "description", "", "Reason for adding shards")
	cmd.MarkFlagRequired("user")
	cmd.MarkFlagRequired("amount")
	return cmd
}

func shardsRemoveCmd() *cobra.Command {
	var userID, amount int
	var description string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove shards from a user's balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Post(fmt.Sprintf("/admin/users/%d/remove-shards", userID), map[string]interface{}{
				"amount":      amount,
				"description": description,
			})
			if err != nil {
				return err
			}

			fmt.Println("Shards removed.")
			PrintObject(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&userID, "user", 0, "User ID")
	cmd.Flags().IntVar(&amount, "amount", 0, "Number of shards to remove")
	cmd.Flags().StringVar(&description, "description", "", "Reason for removing shards")
	cmd.MarkFlagRequired("user")
	cmd.MarkFlagRequired("amount")
	return cmd
}

// ──────────────────────────────────────────────
// pricing
// ──────────────────────────────────────────────

func pricingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pricing",
		Short: "Show room pricing table",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			rows, err := c.GetList("/rooms/pricing")
			if err != nil {
				return err
			}

			fmt.Println("=== Room Pricing ===")
			PrintTable([]Column{
				{"Slots", "slots"},
				{"Weekly", "weekly"},
				{"Monthly", "monthly"},
				{"Quarterly", "quarterly"},
				{"Yearly", "yearly"},
			}, rows)
			return nil
		},
	}
}

// ──────────────────────────────────────────────
// status
// ──────────────────────────────────────────────

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Platform overview (nodes, rooms, players)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NewClient()
			resp, err := c.Get("/admin/monitor/overview")
			if err != nil {
				return err
			}

			// Print top-level summary fields.
			fmt.Println("=== Platform Status ===")
			for _, key := range []string{"total_nodes", "total_rooms", "total_players", "total_users"} {
				if v, ok := resp[key]; ok {
					fmt.Printf("  %-16s %s\n", key+":", fmtVal(v))
				}
			}

			// If there is a nodes array, print it as a table.
			if raw, ok := resp["nodes"]; ok {
				if items, ok := raw.([]interface{}); ok && len(items) > 0 {
					fmt.Println("\n=== Nodes ===")
					rows := make([]map[string]interface{}, 0, len(items))
					for _, item := range items {
						if m, ok := item.(map[string]interface{}); ok {
							rows = append(rows, m)
						}
					}
					PrintTable([]Column{
						{"ID", "id"},
						{"Name", "name"},
						{"Host", "host"},
						{"Rooms", "rooms"},
						{"Players", "players"},
						{"Status", "status"},
					}, rows)
				}
			}

			return nil
		},
	}
}
