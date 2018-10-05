package iap

//{ "keys" : [
// {
// "alg" : "ES256",
// "crv" : "P-256",
// "kid" : "BOIWdQ",
// "kty" : "EC",
// "use" : "sig",
// "x" : "cqFXSp-TUxZN3uuNFKsz82GwmCs3y-d4ZBr74btAQt8",
// "y" : "bqqgY8vyllWut5IfjRpWXx8n413PNRONorSFbl93p98" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "5PuDqQ", "kty" : "EC", "use" : "sig", "x" : "i18fVvEF4SW1EAHabO7lYAbOtJeTuxXv1IQOSdQE_Hg", "y" : "92cL23LzmfAH8EPfgySZqoDhxoSxJYekuF2CsWaxiIY" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "s3nVXQ", "kty" : "EC", "use" : "sig", "x" : "UnyNfx2yYjT7IQUQxcV1HhZ_2qKAacAvvQCslOga0hM", "y" : "sVjk-yHj0JzGyIbbkbHPxUy9QbbnNsRVRwKiZjrWA9w" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "rTlk-g", "kty" : "EC", "use" : "sig", "x" : "sm2pfkO5SYLW7vSv3e-XkKBH6SLrxPL5a0Z2MwWfJXY", "y" : "VxQ0E1s8hMLSAAzJkvN4adV6jee1XLtzZreyL4Z6ke4" }, { "alg" : "ES256", "crv" : "P-256", "kid" : "FAWt5w", "kty" : "EC", "use" : "sig", "x" : "8auUAdTS54HmUuIabrTKvWawxmbs81kdbzQMV_Tae0E", "y" : "IS4Ip_KpyeJZJSa8RM5LF4OTrQbsxOtgyI_gnjzdtD4" } ] }

// JSONWebKey is the IAP JWT json web key to represent the public key of the key used to encrypt the IAP JWT
type JSONWebKey struct {
	Algorithm    string `json:"alg"`
	Curve        string `json:"crv"`
	KeyID        string `json:"kid"`
	KeyType      string `json:"kty"`
	PublicKeyUse string `json:"use"`
	X            string `json:"x"`
	Y            string `json:"y"`
}

// JWKResponse as returned by https://www.gstatic.com/iap/verify/public_key-jwk
type JWKResponse struct {
	Keys []JSONWebKey `json:"keys"`
}
