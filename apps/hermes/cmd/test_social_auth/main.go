package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

type AuthResponse struct {
	User    interface{} `json:"user"`
	Message string      `json:"message"`
}

type UserInfoResponse struct {
	User interface{} `json:"user"`
}

func main() {
	testSocialAuth()
}

func testSocialAuth() {
	baseURL := "http://localhost:8080/api/v1"

	// Test health endpoint
	fmt.Println("Testing health endpoint...")
	resp, err := http.Get(baseURL + "/ping")
	if err != nil {
		fmt.Printf("Error testing ping: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Ping response: %s\n\n", string(body))

	// Test auth endpoints (these will require valid Clerk JWT tokens)
	fmt.Println("Testing auth endpoints...")

	// Test GET /auth/user (requires authentication)
	fmt.Println("Testing GET /auth/user (should return 401 without token)...")
	resp, err = http.Get(baseURL + "/auth/user")
	if err != nil {
		fmt.Printf("Error testing auth endpoint: %v\n", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Auth response (status %d): %s\n\n", resp.StatusCode, string(body))
	}

	// Test POST /auth/user (requires authentication)
	fmt.Println("Testing POST /auth/user (should return 401 without token)...")
	resp, err = http.Post(baseURL+"/auth/user", "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		fmt.Printf("Error testing create user endpoint: %v\n", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Create user response (status %d): %s\n\n", resp.StatusCode, string(body))
	}

	fmt.Println("Basic API tests completed.")
	fmt.Println("\nTo test social auth flows:")
	fmt.Println("1. Set up Clerk frontend with Google, GitHub, and Facebook providers")
	fmt.Println("2. Use Clerk's sign-in components to authenticate")
	fmt.Println("3. Include the JWT token in Authorization header: 'Bearer <token>'")
	fmt.Println("4. Test the following endpoints with valid tokens:")
	fmt.Println("   - GET /api/v1/auth/user (get user info)")
	fmt.Println("   - POST /api/v1/auth/user (create user)")
	fmt.Println("   - PUT /api/v1/auth/user (update user)")
	fmt.Println("   - POST /api/v1/auth/refresh (refresh token)")
	fmt.Println("   - POST /api/v1/auth/logout (logout)")
}