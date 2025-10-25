package mofu

// Common Header Fields

const (
	HeaderAccept             = "Accept"
	HeaderContentType        = "Content-Type"
	HeaderContentLength      = "Content-Length"
	HeaderUserAgent          = "User-Agent"
	HeaderAuthorization      = "Authorization"
	HeaderCacheControl       = "Cache-Control"
	HeaderContentEncoding    = "Content-Encoding"
	HeaderContentDisposition = "Content-Disposition"
	HeaderAcceptEncoding     = "Accept-Encoding"
	HeaderAcceptLanguage     = "Accept-Language"
	HeaderCookie             = "Cookie"
	HeaderSetCookie          = "Set-Cookie"
	HeaderLocation           = "Location"
	HeaderServer             = "Server"
)

// Content Types

const (
	ContentHTML        = "text/html"
	ContentPlain       = "text/plain"
	ContentCSS         = "text/css"
	ContentCSV         = "text/csv"
	ContentTextEvent   = "text/event-stream"
	ContentJSON        = "application/json"
	ContentXML         = "application/xml"
	ContentForm        = "application/x-www-form-urlencoded"
	ContentJavaScript  = "application/javascript"
	ContentOctetStream = "application/octet-stream"
	ContentPDF         = "application/pdf"
	ContentZIP         = "application/zip"
	ContentMultipart   = "multipart/form-data"

	// Images

	ContentSVG  = "image/svg+xml"
	ContentPNG  = "image/png"
	ContentJPEG = "image/jpeg"
	ContentGIF  = "image/gif"
	ContentWebP = "image/webp"
	ContentICO  = "image/x-icon"

	// Charsets

	ContentUTF8  = "charset=utf-8"
	ContentUTF16 = "charset=utf-16"
	ContentASCII = "charset=us-ascii"
)

// Encodings

const (
	EncodingGzip     = "gzip"
	EncodingDeflate  = "deflate"
	EncodingBrotli   = "br"
	EncodingZstd     = "zstd"
	EncodingIdentity = "identity" // no encoding
)

// Accept Types

const (
	AcceptAll   = "*/*"
	AcceptHTML  = "text/html"
	AcceptJSON  = "application/json"
	AcceptXML   = "application/xml"
	AcceptPlain = "text/plain"
)
