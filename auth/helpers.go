package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sethgrid/pester"
)

// getIAPJWKs returns the list of JWKs used by google's IAP from https://www.gstatic.com/iap/verify/public_key-jwk
func getIAPJWKs() (keysResponse *IAPJWKResponse, err error) {

	response, err := pester.Get("https://www.gstatic.com/iap/verify/public_key-jwk")
	if err != nil {
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return keysResponse, fmt.Errorf("https://www.gstatic.com/iap/verify/public_key-jwk responded with status code %v", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &keysResponse)
	if err != nil {
		return
	}

	return
}

var iapJWKs map[string]*ecdsa.PublicKey
var iapJWKLastFetched time.Time

// GetCachedIAPJWK returns IAP json web keys from cache or fetches them from source
func GetCachedIAPJWK(kid string) (jwk *ecdsa.PublicKey, err error) {

	if iapJWKs == nil || iapJWKLastFetched.Add(time.Hour*24).Before(time.Now().UTC()) {

		jwks, err := getIAPJWKs()
		if err != nil {
			return nil, err
		}

		// turn array into map and converto to *ecdsa.PublicKey
		iapJWKs = make(map[string]*ecdsa.PublicKey)
		for _, key := range jwks.Keys {

			x, err := base64.RawURLEncoding.DecodeString(key.X)
			if err != nil {
				return nil, err
			}

			y, err := base64.RawURLEncoding.DecodeString(key.Y)
			if err != nil {
				return nil, err
			}

			publicKey := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(x),
				Y:     new(big.Int).SetBytes(y),
			}

			iapJWKs[key.KeyID] = publicKey
		}

		iapJWKLastFetched = time.Now().UTC()
	}

	if val, ok := iapJWKs[kid]; ok {
		return val, err
	}

	return nil, fmt.Errorf("Key with kid %v does not exist at https://www.gstatic.com/iap/verify/public_key-jwk", kid)
}

func getEmailFromIAPJWT(tokenString string, iapAudience string) (email string, err error) {

	// ensure this uses UTC even though google's servers all run in UTC
	jwt.TimeFunc = time.Now().UTC

	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		// check algorithm is correct
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// get public key for kid
		publicKey, err := GetCachedIAPJWK(token.Header["kid"].(string))
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		// verify audience
		expectedAudience := iapAudience
		actualAudience := claims["aud"].(string)
		if actualAudience != expectedAudience {
			return "", fmt.Errorf("Actual audience %v is not equal to expected audience %v", actualAudience, expectedAudience)
		}

		// verify issuer
		expectedIssuer := "https://cloud.google.com/iap"
		actualIssuer := claims["iss"].(string)
		if actualIssuer != expectedIssuer {
			return "", fmt.Errorf("Actual issuer %v is not equal to expected issuer %v", actualIssuer, expectedIssuer)
		}

		email = claims["email"].(string)
		if email == "" {
			return "", fmt.Errorf("Email is empty")
		}

		return
	}

	return "", fmt.Errorf("Token is not valid")
}

// GetUserFromIAPJWT validates IAP JWT and returns auth.User
func GetUserFromIAPJWT(tokenString string, iapAudience string) (user User, err error) {

	if tokenString == "" {
		return user, fmt.Errorf("IAP jwt is empty")
	}

	email, err := getEmailFromIAPJWT(tokenString, iapAudience)
	if err != nil {
		return user, err
	}

	user = User{
		Authenticated: true,
		Email:         email,
		Provider:      "google",
	}

	return
}

// getGoogleJWKs returns the list of JWKs used by google's apis from https://www.googleapis.com/oauth2/v3/certs
func getGoogleJWKs() (keysResponse *GoogleJWKResponse, err error) {

	response, err := pester.Get("https://www.googleapis.com/oauth2/v3/certs")
	if err != nil {
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return keysResponse, fmt.Errorf("https://www.googleapis.com/oauth2/v3/certs responded with status code %v", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	// unmarshal json body
	err = json.Unmarshal(body, &keysResponse)
	if err != nil {
		return
	}

	return
}

var googleJWKs map[string]*rsa.PublicKey
var googleJWKLastFetched time.Time

// GetCachedGoogleJWK returns google's json web keys from cache or fetches them from source
func GetCachedGoogleJWK(kid string) (jwk *rsa.PublicKey, err error) {

	if googleJWKs == nil || googleJWKLastFetched.Add(time.Hour*24).Before(time.Now().UTC()) {

		jwks, err := getGoogleJWKs()
		if err != nil {
			return nil, err
		}

		// turn array into map and converto to *rsa.PublicKey
		googleJWKs = make(map[string]*rsa.PublicKey)
		for _, key := range jwks.Keys {

			n, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				return nil, err
			}

			e := 0
			// the default exponent is usually 65537, so just compare the base64 for [1,0,1] or [0,1,0,1]
			if key.E == "AQAB" || key.E == "AAEAAQ" {
				e = 65537
			} else {
				return nil, fmt.Errorf("JWK key exponent %v can't be converted to int", key.E)
			}

			publicKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(n),
				E: e,
			}

			googleJWKs[key.KeyID] = publicKey
		}

		googleJWKLastFetched = time.Now().UTC()
	}

	if val, ok := googleJWKs[kid]; ok {
		return val, err
	}

	return nil, fmt.Errorf("Key with kid %v does not exist at https://www.googleapis.com/oauth2/v3/certs", kid)
}

func isValidGoogleJWT(tokenString string) (valid bool, err error) {

	// ensure this uses UTC even though google's servers all run in UTC
	jwt.TimeFunc = time.Now().UTC

	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		// check algorithm is correct
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// get public key for kid
		publicKey, err := GetCachedGoogleJWK(token.Header["kid"].(string))
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})

	if err != nil {
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		// "aud": "https://ci.estafette/io/api/integrations/pubsub/events",
		// "azp": "118094230988819892802",
		// "email": "estafette@estafette.iam.gserviceaccount.com",
		// "email_verified": true,
		// "exp": 1568134507,
		// "iat": 1568130907,
		// "iss": "https://accounts.google.com",
		// "sub": "118094230988819892802"

		// verify issuer
		expectedIssuer := "https://accounts.google.com"
		actualIssuer := claims["iss"].(string)
		if actualIssuer != expectedIssuer {
			return false, fmt.Errorf("Actual issuer %v is not equal to expected issuer %v", actualIssuer, expectedIssuer)
		}

		emailVerified := claims["email_verified"].(bool)
		if !emailVerified {
			return false, fmt.Errorf("Email claim is not verified")
		}

		return true, nil
	}

	return false, fmt.Errorf("Token is not valid")
}
