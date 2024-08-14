package utils

import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Firebase struct {
	Auth      *auth.Client
	Firestore *firestore.Client
}
type CustomClaims struct {
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Iss      string `json:"iss"`
	Aud      string `json:"aud"`
	AuthTime int64  `json:"auth_time"`
	UserId   string `json:"user_id"`
	Sub      string `json:"sub"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
	Email    string `json:"email"`
	jwt.StandardClaims
}

func NewFirebase(ctx context.Context) *Firebase {
	opt := option.WithCredentialsFile("./serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}
	auth, err := app.Auth(ctx)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}
	firestore, err := app.Firestore(ctx)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}
	return &Firebase{
		Auth:      auth,
		Firestore: firestore,
	}
}

func VerifyFirebaseJWT(firebase *Firebase, ctx context.Context, tokenString string) error {
	_, err := firebase.Auth.VerifyIDToken(ctx, tokenString)
	if err != nil {
		return err
	}
	return nil
}
func CheckFirebaseJWT(tokenString string) (CustomClaims, error) {

	resp, err := http.Get("https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com")
	if err != nil {
		log.Fatalf("Failed to make a request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read the response body: %v", err)
	}
	var result map[string]interface{}
	err = json.Unmarshal([]byte(body), &result)

	if err != nil {
		log.Fatalf("Failed to json unmarshal: %v", err)
	}

	parts := strings.Split(tokenString, ".")
	// decode the header
	headerJson, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		fmt.Printf("Error decoding JWT header:", err)
		return CustomClaims{}, err
	}
	var header map[string]interface{}
	err = json.Unmarshal(headerJson, &header)
	if err != nil {
		fmt.Printf("Error unmarshalling JWT header:", err)
		return CustomClaims{}, err
	}
	kid := header["kid"].(string)
	certString := result[kid].(string)
	block, _ := pem.Decode([]byte(certString))
	if block == nil {
		fmt.Printf("failed to parse PEM block containing the public key")
		return CustomClaims{}, err
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("failed to parse certificate", err)
		return CustomClaims{}, err
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	// 署名を検証
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return rsaPublicKey, nil
	})
	if err != nil {
		return CustomClaims{}, errors.New("Token is not valid.")
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if time.Unix(claims.Exp, 0).Before(time.Now()) {
			return CustomClaims{}, errors.New("Token is valid. But token is expired.")
		} else {
			return *claims, nil
		}
	} else {
		return CustomClaims{}, errors.New("Token is not valid")
	}
}
