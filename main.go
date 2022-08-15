package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/viper"
	"github.com/vartanbeno/go-reddit/v2/reddit"
	"io"
	"log"
	"net/http"
	"time"
)

var httpClient = http.DefaultClient

type PoECurrency struct {
	Lines []struct {
		CurrencyTypeName string `json:"currencyTypeName"`
		Pay              struct {
			ID                int       `json:"id"`
			LeagueID          int       `json:"league_id"`
			PayCurrencyID     int       `json:"pay_currency_id"`
			GetCurrencyID     int       `json:"get_currency_id"`
			SampleTimeUtc     time.Time `json:"sample_time_utc"`
			Count             int       `json:"count"`
			Value             float64   `json:"value"`
			DataPointCount    int       `json:"data_point_count"`
			IncludesSecondary bool      `json:"includes_secondary"`
			ListingCount      int       `json:"listing_count"`
		} `json:"pay,omitempty"`
		Receive struct {
			ID                int       `json:"id"`
			LeagueID          int       `json:"league_id"`
			PayCurrencyID     int       `json:"pay_currency_id"`
			GetCurrencyID     int       `json:"get_currency_id"`
			SampleTimeUtc     time.Time `json:"sample_time_utc"`
			Count             int       `json:"count"`
			Value             float64   `json:"value"`
			DataPointCount    int       `json:"data_point_count"`
			IncludesSecondary bool      `json:"includes_secondary"`
			ListingCount      int       `json:"listing_count"`
		} `json:"receive,omitempty"`
		PaySparkLine struct {
			Data        []interface{} `json:"data"`
			TotalChange float64       `json:"totalChange"`
		} `json:"paySparkLine"`
		ReceiveSparkLine struct {
			Data        []float64 `json:"data"`
			TotalChange float64   `json:"totalChange"`
		} `json:"receiveSparkLine"`
		ChaosEquivalent           float64 `json:"chaosEquivalent"`
		LowConfidencePaySparkLine struct {
			Data        []interface{} `json:"data"`
			TotalChange float64       `json:"totalChange"`
		} `json:"lowConfidencePaySparkLine"`
		LowConfidenceReceiveSparkLine struct {
			Data        []float64 `json:"data"`
			TotalChange float64   `json:"totalChange"`
		} `json:"lowConfidenceReceiveSparkLine"`
		DetailsID string `json:"detailsId"`
	} `json:"lines"`
	CurrencyDetails []struct {
		ID      int    `json:"id"`
		Icon    string `json:"icon"`
		Name    string `json:"name"`
		TradeID string `json:"tradeId,omitempty"`
	} `json:"currencyDetails"`
	Language struct {
		Name         string `json:"name"`
		Translations struct {
		} `json:"translations"`
	} `json:"language"`
}

func Fetch(db *pgx.Conn, schema string, table string) (bool, bool) {
	var CurrencyResponse PoECurrency
	var mirrorValue float64
	var exaltValue float64
	var divineValue float64
	var mirrorBool bool
	var exaltBool bool
	DeleteDataAll(db, schema, table)
	req, err := http.NewRequest("GET", "https://poe.ninja/api/data/CurrencyOverview?league=Sentinel&type=Currency&language=en", nil)
	if err != nil {
		panic(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&CurrencyResponse)
	if err != nil {
		panic(err)
	}
	for v := range CurrencyResponse.Lines {
		if CurrencyResponse.Lines[v].CurrencyTypeName == "Exalted Orb" {
			exaltValue = CurrencyResponse.Lines[v].ChaosEquivalent
			AddData(db, "Exalted Orb", schema, table, "Currency")
			UpdateDataFloat(db, schema, table, exaltValue, "ChaosValue", "Currency", "Exalted Orb")
		}
		if CurrencyResponse.Lines[v].CurrencyTypeName == "Divine Orb" {
			divineValue = CurrencyResponse.Lines[v].ChaosEquivalent
			AddData(db, "Divine Orb", schema, table, "Currency")
			UpdateDataFloat(db, schema, table, divineValue, "ChaosValue", "Currency", "Divine Orb")
		}
		if CurrencyResponse.Lines[v].CurrencyTypeName == "Mirror of Kalandra" {
			mirrorValue = CurrencyResponse.Lines[v].ChaosEquivalent
			AddData(db, "Mirror of Kalandra", schema, table, "Currency")
			UpdateDataFloat(db, schema, table, mirrorValue, "ChaosValue", "Currency", "Mirror of Kalandra")
		}
	}
	if divineValue > exaltValue {
		exaltBool = true
	} else {
		exaltBool = false
	}
	if divineValue > mirrorValue {
		mirrorBool = true
	} else {
		mirrorBool = false
	}
	return exaltBool, mirrorBool
}
func FetchCredentials() (string, string, string, string, string, string, string, string) {
	viper.SetConfigName("credentials")
	viper.AddConfigPath(".")
	viper.SetConfigType("env")

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	var botid = viper.GetString("BOTID")
	var botsecret = viper.GetString("BOTSECRET")
	var botusername = viper.GetString("BOTUSERNAME")
	var botpassword = viper.GetString("BOTPASSWORD")
	var serverpassword = viper.GetString("SERVERPASSWORD")
	var serverhost = viper.GetString("SERVERHOST")
	var serverport = viper.GetString("SERVERPORT")
	var serverusername = viper.GetString("SERVERUSERNAME")
	return botid, botsecret, botusername, botpassword, serverpassword, serverhost, serverport, serverusername
}
func bomb(client *reddit.Client, db *pgx.Conn, servicename string) {
	var OptedUsers []string
	var OptedService string
	retrieveOpteds, _, err := goqu.Select("usernames").From("optedusers.usertable").ToSQL()
	if err != nil {
		log.Fatal("failed to build query", err)
	}
	rows, _ := db.Query(context.Background(), retrieveOpteds)
	for rows.Next() {
		var user string
		// Scan the result into the struct.
		err := rows.Scan(&user)
		if err != nil {
			log.Fatal("failed to scan row> ", err)
		}
		OptedUsers = append(OptedUsers, user)
	}
	for user := range OptedUsers {
		retrieveExaltNotifs, _, err := goqu.Select("optedservice").From("optedusers.usertable").Where(goqu.Ex{"usernames": OptedUsers[user]}).ToSQL()
		if err != nil {
			log.Fatal("failed to build query", err)
		}
		rows, _ := db.Query(context.Background(), retrieveExaltNotifs)
		for rows.Next() {
			if rows.Scan(&OptedService) != nil {
				log.Fatal("failed to scan row> ", err)
			}
			if OptedService == servicename {
				send, err := client.Message.Send(context.Background(), &reddit.SendMessageRequest{To: OptedUsers[user], Subject: "DING DONG DING DONG", Text: "Divine Orb is officially higher than Exalted Orbs. Are we... fucked? Or... Are we good? Future will tell my friend."})
				if err != nil {
					return
				}
				if send.StatusCode != 200 {
					continue
				}
			}
		}
	}
}
func FetchOpteds(client *reddit.Client, schema string, table string, db *pgx.Conn) {
	_, messages, resp, err := client.Message.InboxUnread(context.Background(), &reddit.ListOptions{Limit: 50})
	if err != nil || resp.StatusCode != 200 {
		panic(err)
	}
	for _, message := range messages {
		if message.Subject != "" {
			AddData(db, message.Author, schema, table, "usernames")
			// Remove ! from message.Text
			message.Subject = message.Subject[1:]
			UpdateData(db, schema, table, message.Subject, "optedservice", "usernames", message.Author)
		}
		response, err := client.Message.Read(context.Background(), message.FullID)
		if err != nil || response.StatusCode != 200 {
			return
		}
	}
}
func main() {
	// Create reddit client.
	var botid, botsecret, botusername, botpassword, serverpassword, serverhost, serverport, serverusername = FetchCredentials()
	var redditClient, _ = reddit.NewClient(reddit.Credentials{ID: botsecret, Secret: botid, Username: botusername, Password: botpassword})
	var options = "&options=--cluster%3Dpool-gorgon-2847"
	var schema = "optedusers"
	var table = "usertable"
	var currencyschema = "currencies"
	var currencytable = "currencytable" // Setupping database
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/defaultdb?sslmode=verify-full%s", serverusername, serverpassword, serverhost, serverport, options)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	defer func(conn *pgx.Conn, ctx context.Context) {
		err := conn.Close(ctx)
		if err != nil {
			panic(err)
		}
	}(conn, context.Background())
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	// Main loop of the program:
	// Select case is fired with timer every 3 minutes to not fuck with Reddit API.
	// Maximum of 50 unread messages are fetched and processed.
	// poe.ninja is pinged every 1 minutes, I hope they dont get angry at me.
	// Divine>exalt is checked in every fetch, if it is higher... bomb the opted in users messagebox!
	threemintimer := time.NewTimer(time.Minute * 3)
	onemintimer := time.NewTimer(time.Minute * 1)
	for {
		select {
		case <-threemintimer.C:
			FetchOpteds(redditClient, schema, table, conn)
			threemintimer.Reset(time.Minute * 3)
		case <-onemintimer.C:
			fmt.Println("Fetching poe.ninja...")
			exaltBool, mirrorBool := Fetch(conn, currencyschema, currencytable)
			onemintimer.Reset(time.Minute * 1)
			fmt.Println("Done!")
			if exaltBool {
				fmt.Println("Bombing opted in users...")
				// exaltedhigher / Divine Orb is higher than Exalted Orb.
				//	mirrorhigher / Divine Orb is higher than Mirror OF Kalandra
				bomb(redditClient, conn, "exaltedhigher")
				fmt.Println("Done!")
			}
			if mirrorBool {
				fmt.Println("Bombing opted in users...")
				bomb(redditClient, conn, "mirrorhigher")
				fmt.Println("Done!")
			}
		}
	}
}
