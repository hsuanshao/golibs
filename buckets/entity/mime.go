package entity

import "strings"

// ContentType defines file mime type
type ContentType string

// Application content type
const (
	// APK content type
	APK ContentType = "application/vnd.android.package-archive"
	// PDF mime content type
	PDF ContentType = "application/pdf"
	// JSON content type
	JSON ContentType = "application/json"
	// TOML content type
	TOML ContentType = "application/toml"
	// WASM content type (webassembly), .wasm.gz
	WASM ContentType = "application/wasm"
)

// Binary file
const (
	BINType ContentType = "application/octet-stream"
	BINARY  ContentType = "application/binary"
	XBINARY ContentType = "application/x-binary"
)

// Text content type
const (
	// HTML content type
	HTML ContentType = "text/html"
	// TextPlain content type
	TextPlain ContentType = "text/plain"
	// YAML content type
	YAML ContentType = "text/yaml"
	// Calendar is the iCalendar content format (.ics)
	Calendar ContentType = "text/calendar"
	// Javascript content type
	Javascript ContentType = "text/javascript"
	// CSV content type
	CSV ContentType = "text/csv"
	// CSS content type
	CSS ContentType = "text/css"
)

// Image content type
const (
	// ICO image content type
	ICO ContentType = "image/vnd.microsoft.icon"
	// JPG image mime content type
	JPG ContentType = "image/jpeg"
	// GIF image mime content type
	GIF ContentType = "image/gif"
	// WebP image mime content type
	WebP ContentType = "image/webp"
	// PNG image mimi content type
	PNG ContentType = "image/png"
)

// certification content type
const (
	// PKCS8 and the .p8 extension are defined in RFC 5958#section-7.1. The .key extension is Apache mod_ssl practice    .p8  .key
	PKCS8 ContentType = "application/pkcs8"
	// PKCS10 and the .p10 extension are defined in RFC 5967#section-3.1. The .csr extension is Apache mod_ssl practice.
	PKCS10 ContentType = "application/pkcs10" //  .p10 .csr
	//pkix-cert and the .cer extension are defined in RFC 2585#section-4.1
	PKIXCert ContentType = "application/pkix-cert" // .cer
	//pkix-crl and the .crl extension are defined in RFC 2585#section-4.2 as well.
	PKIXCrl ContentType = "application/pkix-crl" // .crl

	// pkcs7-mime and the .p7c extension are defined in RFC 5273#page-3.
	PKCS7mime ContentType = "application/pkcs7-mime" // .p7c

	//x-x509-ca-cert and the .crt extension were introduced by Netscape. File contents are the same as with pkix-cert: a DER encoded X.509 certificate. [RFC 5280#section-4]
	X509CAcert ContentType = "application/x-x509-ca-cert" // .crt .der

	// x-x509-user-cert was also introduced by Netscape. It is used to install certificates into (some) browsers

	X509UserCert ContentType = "application/x-x509-user-cert" // .crt

	// x-pkcs7-crl was introduced by Netscape as well. Note that the .crl extension conflicts with pkix-crl. File contents are the same in either case: a DER encoded X.509 CRL. [RFC 5280#section-5]
	XPKCS7Crl ContentType = "application/x-pkcs7-crl" // .crl

	// x-pem-file and the .pem extension stem from a predecessor of S/MIME: Privacy Enhanced Mail
	PEM ContentType = "application/x-pem-file" // .pem

	// x-pkcs12 and the .p12 extension are used for PKCS#12 files. The .pfx extension is a relic from a predecessor of PKCS#12. It is still used in Microsoft environments (the extension not the format.) .p12 .pfx
	XPKCS17 ContentType = "application/x-pkcs12"

	// x-pkcs7-certificates as well as the .p7b and .spc extensions were introduced by Microsoft. File contents are the same as with pkcs7-mime: a DER encoded certs-only PKCS#7 bundle. [RFC 2315#section-9.1]
	XPKCS17Cert ContentType = "application/x-pkcs7-certificates" // .p7b .spc
	// x-pkcs7-certreqresp and the .p7r extension were also introduced by Microsoft. Likely yet another alias for pkcs7-mime.
	XPJCS7CertReqResp ContentType = "application/x-pkcs7-certreqresp" // .p7r
)

// Video content type
const (
	// AVI video content type
	AVI ContentType = "video/x-msvideo"
	// H261 video content type
	H261 ContentType = "video/h261"
	// H263 video content type
	H263 ContentType = "video/h263"
	// H264 video content type
	H264 ContentType = "video/h264"
	// MP4 video content type
	MP4 ContentType = "video/mp4"
)

func (c ContentType) String() string {
	return string(c)
}

// IsMatchContentType validates that a detected MIME type string is compatible
// with the expected ContentType. It uses three tiers of matching:
//  1. Exact match (e.g. "image/png" == "image/png")
//  2. Prefix match for charset variants (e.g. "application/json; charset=utf-8" starts with "application/json")
//  3. Generic fallback: if detection returns "application/octet-stream" or "text/plain" (with any charset),
//     trust the caller's specification. This is common for text-based formats (JSON, YAML, CSV)
//     that lack distinguishing magic bytes.
func IsMatchContentType(detected string, expected ContentType) bool {
	expectedStr := expected.String()

	// Tier 1: exact match
	if detected == expectedStr {
		return true
	}

	// Tier 2: prefix match (handles charset suffixes)
	if strings.HasPrefix(detected, expectedStr) {
		return true
	}

	// Tier 3: generic detected types — trust the caller
	if detected == "application/octet-stream" || strings.HasPrefix(detected, "text/plain") {
		return true
	}

	return false
}
