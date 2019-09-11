package auth

// {
// 	"keys" : [
// 	   {
// 		  "alg" : "ES256",
// 		  "crv" : "P-256",
// 		  "kid" : "f9R3yg",
// 		  "kty" : "EC",
// 		  "use" : "sig",
// 		  "x" : "SqCmEwytkqG6tL6a2GTQGmSNI4jHYo5MeDUs7DpETVg",
// 		  "y" : "Ql1yyBay4NrGbzasPEhp56Jy6HqoDkqkXYyQRreCOo0"
// 	   },

// IAPJSONWebKey is the IAP JWT json web key to represent the public key of the key used to encrypt the IAP JWT
type IAPJSONWebKey struct {
	Algorithm    string `json:"alg"`
	Curve        string `json:"crv"`
	KeyID        string `json:"kid"`
	KeyType      string `json:"kty"`
	PublicKeyUse string `json:"use"`
	X            string `json:"x"`
	Y            string `json:"y"`
}

// IAPJWKResponse as returned by https://www.gstatic.com/iap/verify/public_key-jwk
type IAPJWKResponse struct {
	Keys []IAPJSONWebKey `json:"keys"`
}

// {
// 	"keys": [
// 	  {
// 		"e": "AQAB",
// 		"kty": "RSA",
// 		"alg": "RS256",
// 		"n": "0QW_fsq8WFtNPeOp8cJO1zoToB_E2HBs1Y4ceJB_3qgJmATBCffGwTm7waYEgIlQbJ7fqP1ttgdab-5yQTDGrE51_KS1_3jlB_EDYZPciT3uzHo69BE0v4h9A29fG2MTR1iwkjqDuWE-JN1TNQUeYZ554WYktX1d0qnaiOhM8jNLcuU948LW9d-9xwd7NwnKD_PakCOWRUqXZVYnS7EsTMG4aZpZk0ZB-695tsH-NmwqISPfXI7sEjINRd2PdD9mvs2xAfp-T7eaCV-C3fTfoHDGB3Vwkfn1rG2p-hFB57vzUYB8vEdRgR8ehhEgWndLU6fovvVToWFnPcvkm-ZFxw",
// 		"use": "sig",
// 		"kid": "05a02649a5b45c90fdfe4da1ebefa9c079ab593e"
// 	  },

// GoogleJSONWebKey is the Google JWT json web key to represent the public key of the key used to encrypt the Google JWT
type GoogleJSONWebKey struct {
	E            string `json:"e"`
	KeyType      string `json:"kty"`
	Algorithm    string `json:"alg"`
	N            string `json:"n"`
	PublicKeyUse string `json:"use"`
	KeyID        string `json:"kid"`
}

// GoogleJWKResponse as returned by https://www.googleapis.com/oauth2/v3/certs
type GoogleJWKResponse struct {
	Keys []GoogleJSONWebKey `json:"keys"`
}

// User has the basic properties used for authentication
type User struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email"`
}
