module github.com/josephalai/sentanyl/core-service

go 1.25.0

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/josephalai/sentanyl/pkg v0.0.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	sentanyl v0.0.0
)

require (
	github.com/antihax/optional v1.0.0 // indirect
	github.com/bradfitz/slice v0.0.0-20180809154707-2b758aa73013 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/getbrevo/brevo-go v1.0.0 // indirect
	github.com/gin-contrib/gzip v0.0.3 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-chi/chi v4.0.0+incompatible // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/gomodule/redigo v1.8.2 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/schema v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mailgun/mailgun-go/v4 v4.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/sfreiberg/gotwilio v0.0.0-20200916182813-169c4cd5c691 // indirect
	github.com/stripe/stripe-go/v82 v82.1.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace (
	github.com/josephalai/sentanyl/pkg => ../pkg
	sentanyl => ../
)
