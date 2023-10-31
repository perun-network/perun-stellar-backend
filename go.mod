module perun.network/perun-stellar-backend

go 1.19

require github.com/stellar/go v0.0.0-20231003185205-facabfc2f4c4

require (
	github.com/stellar/go-xdr v0.0.0-20230919160922-6c7b68458206
	github.com/stretchr/testify v1.8.1
	perun.network/go-perun v0.10.6
	polycry.pt/poly-go v0.0.0-20220222131629-aa4bdbaab60b
)

//replace github.com/stellar/go v0.0.0-20231003185205-facabfc2f4c4 => github.com/perun-network/go v0.0.0-20231003185205-facabfc2f4c4
replace github.com/stellar/go v0.0.0-20231003185205-facabfc2f4c4 => ../stellarfork/go

require (
	github.com/2opremio/pretty v0.2.2-0.20230601220618-e1d5758b2a95 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Masterminds/squirrel v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/adjust/goautoneg v0.0.0-20150426214442-d788f35a0315 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go v1.44.326 // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/creachadair/jrpc2 v0.41.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/getsentry/raven-go v0.0.0-20160805001729-c9d3cc542ad1 // indirect
	github.com/go-chi/chi v4.0.3+incompatible // indirect
	github.com/go-errors/errors v0.0.0-20150906023321-a41850380601 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/schema v1.1.0 // indirect
	github.com/guregu/null v2.1.3-0.20151024101046-79c5bd36b615+incompatible // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.2.0 // indirect
	github.com/magiconair/properties v1.8.0 // indirect
	github.com/manucorporat/sse v0.0.0-20160126180136-ee05b128a739 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/pelletier/go-toml v1.9.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.2.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190117184657-bf6a532e95b1 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/rs/cors v0.0.0-20160617231935-a62a804a8a00 // indirect
	github.com/rs/xhandler v0.0.0-20160618193221-ed27b6fd6521 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20190717103323-87ce952f7079 // indirect
	github.com/segmentio/go-loggly v0.5.1-0.20171222203950-eb91657e62b2 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.1.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/spf13/viper v1.3.2 // indirect
	github.com/stellar/throttled v2.2.3-0.20190823235211-89d75816f59d+incompatible // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/xdrpp/goxdr v0.1.1 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/gorp.v1 v1.7.1 // indirect
	gopkg.in/tylerb/graceful.v1 v1.2.13 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
