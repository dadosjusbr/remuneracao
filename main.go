package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/dadosjusbr/api/docs"
	"github.com/dadosjusbr/api/papi"
	"github.com/dadosjusbr/api/uiapi"
	"github.com/dadosjusbr/storage"
	"github.com/dadosjusbr/storage/repo/database"
	"github.com/dadosjusbr/storage/repo/file_storage"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type config struct {
	Port   int    `envconfig:"PORT"`
	DBUrl  string `envconfig:"MONGODB_URI"`
	DBName string `envconfig:"MONGODB_NAME"`

	AwsS3Bucket string `envconfig:"AWS_S3_BUCKET" required:"true"`
	AwsRegion   string `envconfig:"AWS_REGION" required:"true"`

	// Omited fields
	EnvOmittedFields []string `envconfig:"ENV_OMITTED_FIELDS"`

	// Site env
	DadosJusURL    string `envconfig:"DADOSJUS_URL" required:"true"`
	PackageRepoURL string `envconfig:"PACKAGE_REPO_URL" required:"true"`

	// PostgresDB config
	PgUser     string `envconfig:"PG_USER"`
	PgPassword string `envconfig:"PG_PASSWORD"`
	PgDatabase string `envconfig:"PG_DATABASE"`
	PgHost     string `envconfig:"PG_HOST"`
	PgPort     string `envconfig:"PG_PORT"`

	// Query limit env
	SearchLimit   int `envconfig:"SEARCH_LIMIT"`
	DownloadLimit int `envconfig:"DOWNLOAD_LIMIT"`

	// Newrelic config
	NewRelicApp     string `envconfig:"NEWRELIC_APP_NAME"`
	NewRelicLicense string `envconfig:"NEWRELIC_LICENSE"`
}

var pgS3Client *storage.Client
var loc *time.Location
var conf config

// newClient takes a config struct and creates a client to connect with DB and Cloud5
func newClient(db database.Interface, cloud file_storage.Interface) (*storage.Client, error) {
	client, err := storage.NewClient(db, cloud)
	if err != nil {
		return nil, fmt.Errorf("error creating storage.client: %q", err)
	}
	return client, nil
}

func newPostgresDB(c config) (*database.PostgresDB, error) {
	pgDb, err := database.NewPostgresDB(c.PgUser, c.PgPassword, c.PgDatabase, c.PgHost, c.PgPort)
	if err != nil {
		return nil, fmt.Errorf("error creating postgres DB client: %q", err)
	}
	return pgDb, nil
}

func newS3Client(c config) (*file_storage.S3Client, error) {
	s3Client, err := file_storage.NewS3Client(c.AwsRegion, c.AwsS3Bucket)
	if err != nil {
		return nil, fmt.Errorf("error creating S3 client: %v", err.Error())
	}
	return s3Client, nil
}

//	@title			API do dadosjusbr.org
//	@version		1.0
//	@contact.name	DadosJusBr
//	@contact.url	https://dadosjusbr.org
func main() {
	godotenv.Load() // There is no problem if the .env can not be loaded.
	l, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatal(err.Error())
	}
	loc = l
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatal(err.Error())
	}

	// Criando o client do S3
	s3Client, err := newS3Client(conf)
	if err != nil {
		log.Fatal(err)
	}

	pgDB, err := newPostgresDB(conf)
	if err != nil {
		log.Fatal(err)
	}
	pgS3Client, err = newClient(pgDB, s3Client)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := pgDB.GetConnection()
	if err != nil {
		log.Fatalf("Error connecting to postgres: %v", err)
	}

	fmt.Printf("Going to start listening at port:%d\n", conf.Port)

	e := echo.New()

	e.GET("/", func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusMovedPermanently, "/doc")
	}) // necessário para checagem do beanstalk.

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "ui/dist/",
		Browse: true,
		HTML5:  true,
		Index:  "index.html",
	}))
	e.Static("/static", "templates/assets")
	e.Use(middleware.Logger())

	// Internal API configuration
	uiAPIGroup := e.Group("/uiapi")
	var nr *newrelic.Application
	if os.Getenv("DADOSJUSBR_ENV") == "Prod" {
		if conf.NewRelicApp == "" || conf.NewRelicLicense == "" {
			log.Fatalf("Missing environment variables NEWRELIC_APP_NAME or NEWRELIC_LICENSE")
		}
		nr, err = newrelic.NewApplication(
			newrelic.ConfigAppName(conf.NewRelicApp),
			newrelic.ConfigLicense(conf.NewRelicLicense),
			newrelic.ConfigAppLogForwardingEnabled(true),
		)
		if err != nil {
			log.Fatalf("Error bringin up new relic:%q", err)
		}
		uiAPIGroup.Use(nrecho.Middleware(nr))
		uiAPIGroup.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"https://dadosjusbr.com", "http://dadosjusbr.com", "https://dadosjusbr.org", "http://dadosjusbr.org", "https://dadosjusbr-site-novo.herokuapp.com", "http://dadosjusbr-site-novo.herokuapp.com"},
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderContentLength},
		}))
		log.Println("Using production CORS")
	} else {
		uiAPIGroup.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderContentLength, echo.HeaderAccessControlAllowOrigin},
		}))
	}
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/doc", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	uiApiHandler, err := uiapi.NewHandler(pgS3Client, conn, nr, conf.AwsRegion, conf.AwsS3Bucket, loc, conf.EnvOmittedFields, conf.SearchLimit, conf.DownloadLimit)
	if err != nil {
		log.Fatalf("Error creating uiapi handler: %q", err)
	}
	// Return a summary of an agency. This information will be used in the head of the agency page.
	uiAPIGroup.GET("/v1/orgao/resumo/:orgao/:ano/:mes", uiApiHandler.GetSummaryOfAgency)
	uiAPIGroup.GET("/v2/orgao/resumo/:orgao/:ano/:mes", uiApiHandler.V2GetSummaryOfAgency)
	// TODO: Apagar essa rota (v1) quando o site migrar para a nova rota.
	uiAPIGroup.GET("/v1/orgao/resumo/:orgao", uiApiHandler.GetAnnualSummary)
	uiAPIGroup.GET("/v2/orgao/resumo/:orgao", uiApiHandler.GetAnnualSummary)
	// Return all the salary of a month and year. This will be used in the point chart at the entity page.
	uiAPIGroup.GET("/v1/orgao/salario/:orgao/:ano/:mes", uiApiHandler.GetSalaryOfAgencyMonthYear)
	uiAPIGroup.GET("/v2/orgao/salario/:orgao/:ano/:mes", uiApiHandler.V2GetSalaryOfAgencyMonthYear)
	// Return the total of salary of every month of a year of a agency. The salary is divided in Wage, Perks and Others. This will be used to plot the bars chart at the state page.
	uiAPIGroup.GET("/v1/orgao/totais/:orgao/:ano", uiApiHandler.GetTotalsOfAgencyYear)
	uiAPIGroup.GET("/v2/orgao/totais/:orgao/:ano", uiApiHandler.V2GetTotalsOfAgencyYear)
	// Return basic information of a type or state
	uiAPIGroup.GET("/v1/orgao/:grupo", uiApiHandler.GetBasicInfoOfType)
	uiAPIGroup.GET("/v2/orgao/:grupo", uiApiHandler.V2GetBasicInfoOfType)
	uiAPIGroup.GET("/v1/geral/remuneracao/:ano", uiApiHandler.GetGeneralRemunerationFromYear)
	uiAPIGroup.GET("/v2/geral/remuneracao/:ano", uiApiHandler.V2GetGeneralRemunerationFromYear)
	uiAPIGroup.GET("/v1/geral/resumo", uiApiHandler.GeneralSummaryHandler)
	uiAPIGroup.GET("/v2/geral/resumo", uiApiHandler.GetGeneralSummary)
	// Retorna um conjunto de dados a partir de filtros informados por query params
	uiAPIGroup.GET("/v2/pesquisar", uiApiHandler.SearchByUrl)
	// Baixa um conjunto de dados a partir de filtros informados por query params
	uiAPIGroup.GET("/v2/download", uiApiHandler.DownloadByUrl)
	uiAPIGroup.GET("/v2/readme", uiApiHandler.DownloadReadme)

	apiHandler := papi.NewHandler(pgS3Client, conf.DadosJusURL, conf.PackageRepoURL)
	// Public API configuration
	apiGroup := e.Group("/v1", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderContentLength},
	}))
	// Return agency
	apiGroup.GET("/orgao/:orgao", apiHandler.V1GetAgencyById)
	// Return all agencies
	apiGroup.GET("/orgaos", apiHandler.V1GetAllAgencies)
	// Return MIs by year
	apiGroup.GET("/dados/:orgao/:ano", apiHandler.GetMonthlyInfo)
	// Return MIs by month
	apiGroup.GET("/dados/:orgao/:ano/:mes", apiHandler.GetMonthlyInfo)
	// V2 public api, to be used by the new returned data
	apiGroupV2 := e.Group("/v2", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderContentLength},
	}))
	apiGroupV2.GET("/orgao/:orgao", apiHandler.V2GetAgencyById)
	apiGroupV2.GET("/orgaos", apiHandler.V2GetAllAgencies)
	// Return MIs by year
	apiGroupV2.GET("/dados/:orgao/:ano", apiHandler.GetMonthlyInfosByYear)
	// Return MIs by month
	apiGroupV2.GET("/dados/:orgao/:ano/:mes", apiHandler.V2GetMonthlyInfo)
	// Return agency index information
	apiGroupV2.GET("/indice", apiHandler.V2GetAggregateIndexes)
	apiGroupV2.GET("/indice/:param/:valor", apiHandler.V2GetAggregateIndexesWithParams)
	apiGroupV2.GET("/indice/:param/:valor/:ano", apiHandler.V2GetAggregateIndexesWithParams)
	apiGroupV2.GET("/indice/:param/:valor/:ano/:mes", apiHandler.V2GetAggregateIndexesWithParams)
	apiGroupV2.GET("/indices/:ano", apiHandler.V2GetAggregateIndexesWithParams)
	apiGroupV2.GET("/indices/:ano/:mes", apiHandler.V2GetAggregateIndexesWithParams)
	apiGroupV2.GET("/dados/:orgao", apiHandler.V2GetAllAgencyInformation)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Port),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}
	e.Logger.Fatal(e.StartServer(s))
}
