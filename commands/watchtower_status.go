package operator_commands

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/publicsuffix"
)


const broker_url = "https://api.witnesschain.com"

type PolClaim struct {
	Country string `json:"country"`
	City string `json:"city"`
	Region string `json:"region"`
	Latitude float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Radius float32 `json:"radius"`
}

type PublicKey struct{
	Solana string `json:"solana"`
	Ethereum string `json:"ethereum"`
}

type PreLoginBody struct{
	Publickey string `json:"publicKey"`
	KeyType string `json:"keyType"`
	Role string `json:"role"`
	ProjectName string `json:"projectName"`
	Claims PolClaim `json:"claims"`
	WalletPublicKey PublicKey `json:"walletPublicKey"`
	ClientVersion string `json:"clientVersion"`
}

type PreLoginMessage struct {
	Message string `json:"message"`
}

type PreloginResult struct {
	Result PreLoginMessage `json:"result"`
}

type LoginBody struct {
	Signature string `json:"signature"`
}

type LoginResponse struct {
	Result LoginResult `json:"result"`
}

type LoginResult struct {
	Success bool `json:"success"`
}

type ChallengerParameters struct {
	Id string `json:"id"`
}

type ChallengerResponse struct {
	Result ChallengerResult `json:"result"`
}

type ChallengerResult struct {
	Id string `json:"id"`
	LastAlive string `json:"last_alive"`
}

func WatchtowerStatusCmd() *cli.Command {
	return &cli.Command{
		Name:  "watchtowerStatus",
		Usage: "Find status of watchtower",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "watchtower-address",
				Required: true,
			},
			&cli.BoolFlag{
				Name: "pob",

			},
			&cli.BoolFlag{
				Name: "pol",
			},

		},
		Action: func(cCtx *cli.Context) error {
			WatchtowerStatus(cCtx.String("watchtower-address"))
			return nil
		},
	}
}


func WatchtowerStatus(watchtowerAddress string) {
	
	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		log.Fatal(err)
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
			log.Fatal(err)
		}

	client := http.Client{
		Jar: jar,
	}

	message := prelogin(client, privateKey)
	login(client, privateKey, message)
	challenger(client, "IPv4/" + watchtowerAddress)
	challenger(client, "IPv6/" + watchtowerAddress)



}

func prelogin(client http.Client, privateKey *ecdsa.PrivateKey) string {

	loginAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	log.SetFlags(log.LstdFlags | log.Lshortfile) 
	pre_login_body := PreLoginBody{
		KeyType: "ethereum",
		Role: "prover",
		ProjectName: "witness",
		ClientVersion: "99999999999",
		Publickey: loginAddress,
		WalletPublicKey: PublicKey{
			Ethereum: loginAddress,
		},
		Claims: PolClaim{
			Latitude: 0.0,
			Longitude: 0.0,
			Radius: 1000000.0,
			Country: "IN",
			City: "Bengaluru",
			Region: "Karnataka",
		},
	}
	
	body, err:= json.Marshal(pre_login_body)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Post("https://api.witnesschain.com/proof/v1/pol/pre-login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	responseBody, err := io.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	result := PreloginResult{}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result.Result.Message
}

func login(client http.Client, privateKey *ecdsa.PrivateKey, message string) bool{
	signature := signMessage(privateKey, message)

	loginBody := LoginBody{
		Signature: signature,
	}

	loginBodyBytes, err := json.Marshal(loginBody)
	
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Post("https://api.witnesschain.com/proof/v1/pol/login", "application/json", bytes.NewBuffer(loginBodyBytes))
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	loginResponse := LoginResponse{}
	json.Unmarshal(body, &loginResponse)
	if loginResponse.Result.Success != true {
		log.Fatal("unable to login to witnesschain")
	}
	return loginResponse.Result.Success
}

func challenger(client http.Client, id string) {
	challengerParameters := ChallengerParameters{
		Id: id,
	}

	challengerParametersBytes, err := json.Marshal(challengerParameters)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Post("https://api.witnesschain.com/proof/v1/pol/challenger", "application/json", bytes.NewBuffer(challengerParametersBytes))

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	challengerResponse := ChallengerResponse{}
	json.Unmarshal(body, &challengerResponse)

	if len(challengerResponse.Result.LastAlive) == 0 {
		fmt.Print("Watchtower: ", id, " has not login into witnesschain\n")
		return
	}

	last_alive, err := time.Parse(time.RFC3339, challengerResponse.Result.LastAlive)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("watchtower: ", id, " last alive: ", time.Now().Sub(last_alive).Round(1000000000), " ago\n")
}

func signMessage(privateKey *ecdsa.PrivateKey, message string) (string) {

	prefixed_msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed_msg))
	sig, _ := crypto.Sign(hash.Bytes(), privateKey)
	sig[64] += 27
	result := "0x" + hex.EncodeToString(sig)
	return result
}