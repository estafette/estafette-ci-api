package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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

// getIAPJsonWebKeys returns the list of JWKs used by google's IAP from https://www.gstatic.com/iap/verify/public_key-jwk
func getIAPJsonWebKeys() (keysResponse *JWKResponse, err error) {

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

var iapJSONWebKeys map[string]*ecdsa.PublicKey
var jwkLastFetched time.Time

// GetCachedIAPJsonWebKey returns IAP json web keys from cache or fetches them from source
func GetCachedIAPJsonWebKey(kid string) (jwk *ecdsa.PublicKey, err error) {

	if iapJSONWebKeys == nil || jwkLastFetched.Add(time.Hour*24).Before(time.Now().UTC()) {

		jwks, err := getIAPJsonWebKeys()
		if err != nil {
			return nil, err
		}

		// turn array into map and converto to *ecdsa.PublicKey
		iapJSONWebKeys = make(map[string]*ecdsa.PublicKey)
		for _, key := range jwks.Keys {

			xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
			if err != nil {
				return nil, err
			}

			x := new(big.Int)
			x.SetBytes(xBytes)

			yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
			if err != nil {
				return nil, err
			}

			y := new(big.Int)
			y.SetBytes(yBytes)

			publicKey := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			}

			iapJSONWebKeys[key.KeyID] = publicKey
		}

		jwkLastFetched = time.Now().UTC()
	}

	if val, ok := iapJSONWebKeys[kid]; ok {
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
		publicKey, err := GetCachedIAPJsonWebKey(token.Header["kid"].(string))
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
	}

	return
}
