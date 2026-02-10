package brokers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hft/internal/config"
	"hft/internal/http"
	"hft/internal/storage/sqlite"
	"hft/pkg/types"
	"log"
	"time"

	"github.com/golang-jwt/jwt"
)

var FyersAccessToken string
var FyersRefreshToken string
var FyersExpiry int64
var FyersRefreshTokenExpiry int64
var FyersBroker *types.Broker

// ensureValidAccessToken checks if the current access token is expired and refreshes it if needed.
// Returns true if a valid access token is available, false otherwise.
func ensureValidAccessToken() bool {
	// Check if access token is expired
	if time.Now().Unix() >= FyersExpiry {
		if FyersRefreshToken == "" {
			log.Println("[fyers] access token expired and refresh token missing; please log in again via /broker/fyers/callback")
			return false
		}

		newToken := RefreshAccessToken()
		if newToken == "" {
			// Refresh failed (e.g. refresh token expired); user must log in again.
			return false
		}
		// Token refreshed successfully
		return true
	}
	// Token is still valid
	return true
}

// decodeAccessTokenExpiry extracts the expiry (exp) from an access token JWT.
// Returns 0 if the claim is missing or cannot be parsed.
func decodeAccessTokenExpiry(token string) int64 {
	claims, _ := JwtDecode(token)
	if claims == nil {
		return 0
	}
	if exp, ok := claims["exp"].(float64); ok {
		return int64(exp)
	}
	return 0
}

// decodeRefreshTokenExpiry extracts the expiry (exp) from a refresh token JWT.
// If the claim is missing or cannot be parsed, it falls back to an existing
// expiry (if provided) or defaults to 1 year from now.
func decodeRefreshTokenExpiry(token string, existing int64) int64 {
	claims, _ := JwtDecode(token)
	if claims != nil {
		if exp, ok := claims["exp"].(float64); ok {
			return int64(exp)
		}
	}

	// Fallback: keep existing expiry if we have one, otherwise default to 1 year.
	if existing > 0 {
		log.Println("[fyers] refresh token expiry missing, using existing expiry")
		return existing
	}

	log.Println("[fyers] refresh token expiry missing, defaulting to 1 year from now")
	return time.Now().Add(365 * 24 * time.Hour).Unix()
}

func Init() {
	db := sqlite.DefaultDB()
	accessToken, refreshToken, expiry, refreshTokenExpiry := db.Tokens.Get()

	// No stored tokens yet – user needs to log in once via auth flow.
	if accessToken == "" || refreshToken == "" || expiry == 0 {
		log.Println("[fyers] no stored tokens found; please log in via /broker/fyers/callback")
		return
	}

	FyersAccessToken = accessToken
	FyersRefreshToken = refreshToken
	FyersExpiry = expiry
	FyersRefreshTokenExpiry = refreshTokenExpiry

	// Ensure we have a valid access token (refresh if expired)
	if !ensureValidAccessToken() {
		return
	}

	// With a valid token in place, eagerly fetch margin so callers see data immediately.
	GetMargin()
}

func LoginURL(cfg *config.Config) string {
	appId := cfg.Broker[0].Fyers.AppID
	redirectURI := cfg.Broker[0].Fyers.RedirectURI
	return fmt.Sprintf(
		"https://api-t1.fyers.in/api/v3/generate-authcode?client_id=%s&redirect_uri=%s&response_type=code&state=algo",
		appId,
		redirectURI,
	)
}

func GetBroker() *types.Broker {
	return FyersBroker
}

func Connect(authCode string) {
	cfg := config.GlobalConfig
	appId := cfg.Broker[0].Fyers.AppID
	appSecret := cfg.Broker[0].Fyers.AppSecret

	grantType := "authorization_code"
	appIdHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", appId, appSecret)))
	appIdHashString := hex.EncodeToString(appIdHash[:])

	body := fmt.Sprintf(`{
		"grant_type":"%s",
		"appIdHash":"%s",
		"code":"%s"
	}`, grantType, appIdHashString, authCode)

	client := http.DefaultClient()
	responseData, err := client.PostJSONRaw("https://api-t1.fyers.in/api/v3/validate-authcode", body, nil)
	if err != nil {
		log.Fatalf("validate authcode: %v", err)
	}

	FyersAccessToken = responseData["access_token"].(string)
	FyersRefreshToken = responseData["refresh_token"].(string)

	// Decode and cache expiries.
	FyersExpiry = decodeAccessTokenExpiry(FyersAccessToken)
	FyersRefreshTokenExpiry = decodeRefreshTokenExpiry(FyersRefreshToken, 0)

	// Store tokens in database.
	db := sqlite.DefaultDB()
	db.Tokens.Create(FyersAccessToken, FyersRefreshToken, FyersExpiry, FyersRefreshTokenExpiry)
}

func JwtDecode(token string) (map[string]interface{}, error) {
	decodedToken, _ := jwt.Parse(token, nil)
	return decodedToken.Claims.(jwt.MapClaims), nil
}

func GetFyersAccessToken() string {
	if !ensureValidAccessToken() {
		return ""
	}
	return FyersAccessToken
}

func GetFyersRefreshToken() string {
	return FyersRefreshToken
}

func RefreshAccessToken() string {
	cfg := config.GlobalConfig
	appId := cfg.Broker[0].Fyers.AppID
	appSecret := cfg.Broker[0].Fyers.AppSecret

	grantType := "refresh_token"
	appIdHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", appId, appSecret)))
	appIdHashString := hex.EncodeToString(appIdHash[:])
	body := fmt.Sprintf(`{
		"grant_type":"%s",
		"appIdHash":"%s",
		"refresh_token":"%s",
		"pin":"%s"
	}`, grantType, appIdHashString, FyersRefreshToken, cfg.Broker[0].Fyers.Pin)

	client := http.DefaultClient()
	responseData, err := client.PostJSONRaw("https://api-t1.fyers.in/api/v3/validate-refresh-token", body, nil)
	if err != nil {
		log.Printf("[fyers] refresh access token failed: %v; please log in again via /broker/fyers/callback", err)
		return ""
	}

	// If API indicates an error or no access_token, treat refresh as failed (likely refresh token expired).
	rawAccess, ok := responseData["access_token"]
	if !ok {
		log.Printf("[fyers] refresh token appears invalid/expired (no access_token in response); please log in again via /broker/fyers/callback. Response: %+v", responseData)
		return ""
	}

	accessToken, ok := rawAccess.(string)
	if !ok || accessToken == "" {
		log.Printf("[fyers] refresh token appears invalid/expired (bad access_token type); please log in again via /broker/fyers/callback. Response: %+v", responseData)
		return ""
	}

	FyersAccessToken = accessToken
	FyersExpiry = decodeAccessTokenExpiry(FyersAccessToken)

	// Check if refresh token is also returned in the response (some APIs return a new refresh token)
	rawRefresh, hasRefresh := responseData["refresh_token"]
	if hasRefresh {
		if refreshTokenStr, ok := rawRefresh.(string); ok && refreshTokenStr != "" {
			FyersRefreshToken = refreshTokenStr
			FyersRefreshTokenExpiry = decodeRefreshTokenExpiry(FyersRefreshToken, FyersRefreshTokenExpiry)

			// Store both tokens in database.
			db := sqlite.DefaultDB()
			db.Tokens.UpdateWithRefreshToken(FyersAccessToken, FyersRefreshToken, FyersExpiry, FyersRefreshTokenExpiry)
			return FyersAccessToken
		}
	}

	// Only access token was updated, refresh token remains the same.
	// Store token in database.
	db := sqlite.DefaultDB()
	db.Tokens.Update(FyersAccessToken, FyersExpiry)
	return FyersAccessToken
}

func GetMargin() {
	accessToken := GetFyersAccessToken()
	if accessToken == "" {
		log.Println("[fyers] cannot fetch margin: no valid access token; please log in again via /broker/fyers/callback")
		return
	}

	client := http.DefaultClient()
	headers := map[string]string{
		"Authorization": fmt.Sprintf("app_id:%s", accessToken),
	}
	responseData, err := client.GetJSON("https://api-t1.fyers.in/api/v3/funds", headers)
	if err != nil {
		log.Fatalf("get margin: %v", err)
	}

	// Parse Fyers fund response and load to Broker struct
	fundLimits, ok := responseData["fund_limit"].([]interface{})
	if !ok || len(fundLimits) == 0 {
		log.Fatalf("unexpected or missing 'fund_limit' data in response: %v", responseData)
	}

	var (
		totalBalance     float64
		utilizedAmount   float64
		clearBalance     float64
		availableBalance float64
	)

	// Title mappings from API doc/sample:
	// Total Balance         -> Margin/Equity
	// Utilized Amount       -> UtilizedMargin
	// Clear Balance         -> FreeMargin
	// Available Balance     -> AvailableMargin (could be "Clear Balance" or another field if present)

	for _, entry := range fundLimits {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := entryMap["title"].(string)
		equityAmount, _ := entryMap["equityAmount"].(float64)

		switch title {
		case "Total Balance":
			totalBalance = equityAmount
		case "Utilized Amount":
			utilizedAmount = equityAmount
		case "Clear Balance":
			clearBalance = equityAmount
		case "Available Balance": // safeguard if present in future API versions
			availableBalance = equityAmount
		}
	}

	// If AvailableBalance wasn't present, fallback to ClearBalance as 'free margin' (per Fyers docs)
	if availableBalance == 0 {
		availableBalance = clearBalance
	}

	FyersBroker = &types.Broker{
		Name:            "Fyers",
		Margin:          totalBalance,
		Equity:          totalBalance,
		FreeMargin:      clearBalance,
		UtilizedMargin:  utilizedAmount,
		AvailableMargin: availableBalance,
	}
}

func LoadHistory(symbol string, resolution int, from time.Time, to time.Time) []types.Tick {
	// curl --location --request GET 'https://api-t1.fyers.in/data/history?symbol=NSE:SBIN-EQ&resolution=30&date_format=1&range_from=2021-01-01&range_to=2021-01-02&cont_flag=' \
	//  --header 'Authorization:app_id:access_token'
	fromEpoch := from.Unix()
	toEpoch := to.Unix()
	accessToken := GetFyersAccessToken()
	if accessToken == "" {
		log.Println("[fyers] cannot fetch history: no valid access token; please log in again via /broker/fyers/callback")
		return []types.Tick{}
	}

	client := http.DefaultClient()
	headers := map[string]string{
		"Authorization": fmt.Sprintf("app_id:%s", accessToken),
	}
	url := fmt.Sprintf("https://api-t1.fyers.in/data/history?symbol=%s&resolution=%d&date_format=0&range_from=%d&range_to=%d&cont_flag=", symbol, resolution, fromEpoch, toEpoch)
	responseData, err := client.GetJSON(url, headers)
	if err != nil {
		log.Printf("get history: %v", err)
		return []types.Tick{}
	}

	candles, _ := responseData["candles"].([]interface{})
	var ticks []types.Tick

	// 	Candles data containing array of following data for particular time stamp:
	// 1.Current epoch time
	// 2. Open Value
	// 3.Highest Value
	// 4.Lowest Value
	// 5.Close Value
	// 6.Total traded quantity (volume)

	for _, candle := range candles {
		candleArray, _ := candle.([]interface{})
		timestamp, _ := candleArray[0].(float64) // Fyers returns timestamp as float64
		open, _ := candleArray[1].(float64)
		high, _ := candleArray[2].(float64)
		low, _ := candleArray[3].(float64)
		close, _ := candleArray[4].(float64)
		volume, _ := candleArray[5].(float64)

		timestampInt := int64(timestamp)
		ticks = append(ticks, types.Tick{
			Timestamp: time.Unix(timestampInt, 0),
			Time:      timestampInt,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	return ticks
}
