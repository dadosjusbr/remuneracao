package uiapi

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dadosjusbr/proto/coleta"
	"github.com/dadosjusbr/storage"
	"github.com/dadosjusbr/storage/models"
	"github.com/dadosjusbr/storage/repo/database"
	"github.com/dadosjusbr/storage/repo/file_storage"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var loc *time.Location
var hand handler

func TestMain(m *testing.M) {
	l, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatal(err.Error())
	}
	loc = l
	exitValue := m.Run()
	os.Exit(exitValue)
}

func TestGetSummaryOfAgency(t *testing.T) {
	tests := getSummaryOfAgency{}
	t.Run("Test GetSummaryOfAgency when data exists", tests.testWhenDataExists)
	t.Run("Test GetSummaryOfAgency when data does not exist", tests.testWhenDataDoesNotExist)
	t.Run("Test GetSummaryOfAgency when year is invalid", tests.testWhenYearIsInvalid)
	t.Run("Test GetSummaryOfAgency when month is invalid", tests.testWhenMonthIsInvalid)
}

type getSummaryOfAgency struct{}

func (g getSummaryOfAgency) testWhenDataExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agmi := agencyMonthlyInfos()
	agency := &models.Agency{
		ID:            "tjal",
		Name:          "Tribunal de Justiça do Estado de Alagoas",
		Type:          "Estadual",
		Entity:        "Tribunal",
		UF:            "AL",
		TwitterHandle: "TJALagoas",
		OmbudsmanURL:  "TJALagoas.com.br",
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOMA(1, 2020, "tjal").Return(&agmi[0], agency, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/resumo/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSummaryOfAgency(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"orgao": "Tribunal de Justiça do Estado de Alagoas",
			"remuneracao_base": 7.099024400000013e+06,
			"max_remuneracao": 35462.22,
			"max_remuneracao_base": 35462.22,
			"outras_remuneracoes": 1.9515865600000022e+06,
			"max_outras_remuneracoes": 45200.05,
			"descontos": 2221879.66,
  			"max_descontos": 23118.190000000002,
			"timestamp": {
				"seconds": 1,
				"nanos": 1
			},
			"total_membros": 214,
			"total_remuneracao": 7.099024400000013e+06,
			"tem_proximo": true,
			"tem_anterior": true,
			"resumo_rubricas": {
				"auxilio_alimentacao": 100,
        		"licenca_premio": 150,
				"indenizacao_de_ferias": 130,
				"ferias": 220,
				"gratificacao_natalina": 120,
				"licenca_compensatoria": 120,
				"auxilio_saude": 300,
        		"outras": 200
			}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getSummaryOfAgency) testWhenDataDoesNotExist(t *testing.T) {
	mockCrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCrl)
	fsMock := file_storage.NewMockInterface(mockCrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOMA(1, 2020, "tjal").Return(nil, nil, fmt.Errorf("there is no data with this parameters"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/resumo/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSummaryOfAgency(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020, mês=1 ou nome do orgão=tjal são inválidos"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSummaryOfAgency) testWhenYearIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/resumo/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020a", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSummaryOfAgency(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSummaryOfAgency) testWhenMonthIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/resumo/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1a")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSummaryOfAgency(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro mês=1a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func TestGetSalaryOfAgencyMonthYear(t *testing.T) {
	g := getSalaryOfAgencyMonthYear{}
	t.Run("Test when data exists", g.testWhenDataExists)
	t.Run("Test when data does not exist", g.testWhenDataDoesNotExist)
	t.Run("Test when year is invalid", g.testWhenYearIsInvalid)
	t.Run("Test when month is invalid", g.testWhenMonthIsInvalid)
	t.Run("Test when procinfo is not null", g.testWhenProcInfoIsNotNull)
}

type getSalaryOfAgencyMonthYear struct{}

func (g getSalaryOfAgencyMonthYear) testWhenDataExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agmi := agencyMonthlyInfos()
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOMA(1, 2020, "tjal").Return(&agmi[0], nil, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/salario/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSalaryOfAgencyMonthYear(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"max_salario": 35462.22,
			"histograma": {
				"-1": 0,
				"10000": 1,
				"20000": 0,
				"30000": 3,
				"40000": 210,
				"50000": 0
			},
			"package": {
				"url": "https://dadosjusbr.org/download/tjal/datapackage/tjal-2020-1.zip",
				"hash": "4d7ca8986101673aea060ac1d8e5a529",
				"size": 30195
			}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSalaryOfAgencyMonthYear) testWhenDataDoesNotExist(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOMA(1, 2020, "tjal").Return(nil, nil, fmt.Errorf("there is no data with this parameters"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/salario/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSalaryOfAgencyMonthYear(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020, mês=1 ou nome do orgão=tjal são inválidos"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSalaryOfAgencyMonthYear) testWhenYearIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/salario/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020a", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSalaryOfAgencyMonthYear(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSalaryOfAgencyMonthYear) testWhenMonthIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/salario/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1a")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSalaryOfAgencyMonthYear(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro mês=1a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getSalaryOfAgencyMonthYear) testWhenProcInfoIsNotNull(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agmi := agencyMonthlyInfos()
	agmi[0].ProcInfo = &coleta.ProcInfo{
		Stderr: "stderr",
		Cmd:    "docker run ...",
		CmdDir: "/tmp",
		Status: 4,
		Env:    []string{"VAR1=1", "VAR2=2"},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOMA(1, 2020, "tjal").Return(&agmi[0], nil, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/salario/:orgao/:ano/:mes",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano", "mes")
	ctx.SetParamValues("tjal", "2020", "1")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetSalaryOfAgencyMonthYear(ctx)

	expectedCode := http.StatusPartialContent
	expectedJson := `
		{
			"proc_info": {
				"stderr": "stderr",
				"cmd": "docker run ...",
				"cmd_dir": "/tmp",
				"status": 4,
				"env": [
					"VAR1=1",
					"VAR2=2"
				]
			},
			"timestamp": {
				"seconds": 1,
				"nanos": 1
			}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func TestGetBasicInfoOfType(t *testing.T) {
	tests := getBasicInfoOfType{}
	t.Run("Test GetBasicInfoOfType when group is a jurisdiction", tests.testWhenGroupIsAJurisdiction)
	t.Run("Test GetBasicInfoOfType when group is an UF", tests.testWhenGroupIsAnUF)
	t.Run("Test GetBasicInfoOfType when group does not exist", tests.testWhenGroupDoesNotExist)
	t.Run("Test GetBasicInfoOfType when jurisdiction is in irregular case", tests.testWhenJurisdictionIsInIrregularCase)
	t.Run("Test GetBasicInfoOfType when UF is in irregular case", tests.testWhenUFIsInIrregularCase)
}

type getBasicInfoOfType struct{}

func (g getBasicInfoOfType) testWhenGroupIsAJurisdiction(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agencies := []models.Agency{
		{
			ID:     "tjpb",
			Name:   "Tribunal de Justiça da Paraíba",
			Entity: "Tribunal",
		},
		{
			ID:     "tjpe",
			Name:   "Tribunal de Justiça de Pernambuco",
			Entity: "Tribunal",
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOPJ("Estadual").Return(agencies, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/:grupo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("grupo")
	ctx.SetParamValues("justica-estadual")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetBasicInfoOfType(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"grupo": "JUSTICA-ESTADUAL",
			"orgaos": [
				{
					"id_orgao": "tjpb",
					"nome": "Tribunal de Justiça da Paraíba",
					"entidade": "Tribunal"
				},
				{
					"id_orgao": "tjpe",
					"nome": "Tribunal de Justiça de Pernambuco",
					"entidade": "Tribunal"
				}
			]
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getBasicInfoOfType) testWhenGroupIsAnUF(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agencies := []models.Agency{
		{
			ID:     "tjpb",
			Name:   "Tribunal de Justiça da Paraíba",
			Entity: "Tribunal",
		},
		{
			ID:     "mppb",
			Name:   "Ministério Público da Paraíba",
			Entity: "Ministério",
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetStateAgencies("PB").Return(agencies, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/:grupo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("grupo")
	ctx.SetParamValues("PB")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetBasicInfoOfType(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"grupo": "PB",
			"orgaos": [
				{
					"id_orgao": "tjpb",
					"nome": "Tribunal de Justiça da Paraíba",
					"entidade": "Tribunal"
				},
				{
					"id_orgao": "mppb",
					"nome": "Ministério Público da Paraíba",
					"entidade": "Ministério"
				}
			]
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getBasicInfoOfType) testWhenGroupDoesNotExist(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/:grupo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("grupo")
	ctx.SetParamValues("grupo-que-nao-existe")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetBasicInfoOfType(ctx)

	expectedCode := http.StatusNotFound
	expectedJson := `"Grupo não encontrado: 'grupo-que-nao-existe'"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getBasicInfoOfType) testWhenJurisdictionIsInIrregularCase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agencies := []models.Agency{
		{
			ID:     "tjpb",
			Name:   "Tribunal de Justiça da Paraíba",
			Entity: "Tribunal",
		},
		{
			ID:     "tjpe",
			Name:   "Tribunal de Justiça de Pernambuco",
			Entity: "Tribunal",
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetOPJ("Estadual").Return(agencies, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/:grupo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("grupo")
	ctx.SetParamValues("JuStiCa-esTaDuaL")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetBasicInfoOfType(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"grupo": "JUSTICA-ESTADUAL",
			"orgaos": [
				{
					"id_orgao": "tjpb",
					"nome": "Tribunal de Justiça da Paraíba",
					"entidade": "Tribunal"
				},
				{
					"id_orgao": "tjpe",
					"nome": "Tribunal de Justiça de Pernambuco",
					"entidade": "Tribunal"
				}
			]
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getBasicInfoOfType) testWhenUFIsInIrregularCase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agencies := []models.Agency{
		{
			ID:     "tjpb",
			Name:   "Tribunal de Justiça da Paraíba",
			Entity: "Tribunal",
		},
		{
			ID:     "mppb",
			Name:   "Ministério Público da Paraíba",
			Entity: "Ministério",
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetStateAgencies("PB").Return(agencies, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/orgao/:grupo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("grupo")
	ctx.SetParamValues("pB")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetBasicInfoOfType(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"grupo": "PB",
			"orgaos": [
				{
					"id_orgao": "tjpb",
					"nome": "Tribunal de Justiça da Paraíba",
					"entidade": "Tribunal"
				},
				{
					"id_orgao": "mppb",
					"nome": "Ministério Público da Paraíba",
					"entidade": "Ministério"
				}
			]
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func TestGetGeneralSummary(t *testing.T) {
	tests := getGeneralSummary{}
	t.Run("Test GetGeneralSummary when all data are returned", tests.testWhenAllDataAreReturned)
	t.Run("Test GetGeneralSummary when GetAgenciesCount() returns an error", tests.testWhenGetAgenciesCountReturnsAnError)
	t.Run("Test GetGeneralSummary when GetNumberOfMonthsCollected() returns an error", tests.testWhenGetNumberOfMonthsCollectedReturnsAnError)
	t.Run("Test GetGeneralSummary when GetFirstDateWithMonthlyInfo() returns an error", tests.testWhenGetFirstDateWithMonthlyInfoReturnsAnError)
	t.Run("Test GetGeneralSummary when GetLastDateWithMonthlyInfo() returns an error", tests.testWhenGetLastDateWithMonthlyInfoReturnsAnError)
	t.Run("Test GetGeneralSummary when GetGeneralMonthlyInfo() returns an error", tests.testWhenGetGeneralMonthlyInfoReturnsAnError)
}

type getGeneralSummary struct{}

func (g getGeneralSummary) testWhenAllDataAreReturned(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(5, nil)
	dbMock.EXPECT().GetNumberOfMonthsCollected().Return(10, nil)
	dbMock.EXPECT().GetFirstDateWithMonthlyInfo().Return(1, 2018, nil)
	dbMock.EXPECT().GetLastDateWithMonthlyInfo().Return(1, 2023, nil)
	dbMock.EXPECT().GetGeneralMonthlyInfo().Return(float64(1000), nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"num_orgaos": 5,
			"num_meses_coletados": 10,
			"data_inicio": "2018-01-01T22:00:00-02:00",
			"data_fim": "2023-01-01T21:00:00-03:00",
			"remuneracao_total": 1000
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getGeneralSummary) testWhenGetAgenciesCountReturnsAnError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(0, fmt.Errorf("error"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Erro ao contar orgãos: \"error\""`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getGeneralSummary) testWhenGetNumberOfMonthsCollectedReturnsAnError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(5, nil)
	dbMock.EXPECT().GetNumberOfMonthsCollected().Return(0, fmt.Errorf("error"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Erro ao contar registros de meses coletados: \"error\""`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getGeneralSummary) testWhenGetFirstDateWithMonthlyInfoReturnsAnError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(5, nil)
	dbMock.EXPECT().GetNumberOfMonthsCollected().Return(10, nil)
	dbMock.EXPECT().GetFirstDateWithMonthlyInfo().Return(0, 0, fmt.Errorf("error"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Erro buscando primeiro registro de remuneração: \"error\""`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getGeneralSummary) testWhenGetLastDateWithMonthlyInfoReturnsAnError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(5, nil)
	dbMock.EXPECT().GetNumberOfMonthsCollected().Return(10, nil)
	dbMock.EXPECT().GetFirstDateWithMonthlyInfo().Return(2020, 1, nil)
	dbMock.EXPECT().GetLastDateWithMonthlyInfo().Return(0, 0, fmt.Errorf("error"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Erro buscando último registro de remuneração: \"error\""`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getGeneralSummary) testWhenGetGeneralMonthlyInfoReturnsAnError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgenciesCount().Return(5, nil)
	dbMock.EXPECT().GetNumberOfMonthsCollected().Return(10, nil)
	dbMock.EXPECT().GetFirstDateWithMonthlyInfo().Return(2020, 1, nil)
	dbMock.EXPECT().GetLastDateWithMonthlyInfo().Return(2020, 1, nil)
	dbMock.EXPECT().GetGeneralMonthlyInfo().Return(float64(0), fmt.Errorf("error"))

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/geral/resumo",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetGeneralSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Erro buscando valor total de remuneração: \"error\""`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func TestGetGeneralRemunerationFromYear(t *testing.T) {
	tests := getGenerealRemunerationFromYear{}
	t.Run("Test GetGeneralRemunerationFromYear when data exists", tests.testWhenDataExists)
	t.Run("Test GetGeneralRemunerationFromYear when data does not exist", tests.testWhenDataDoesNotExist)
	t.Run("Test GetGeneralRemunerationFromYear when year is invalid", tests.testWhenYearIsInvalid)
}

type getGenerealRemunerationFromYear struct{}

func (g getGenerealRemunerationFromYear) testWhenDataExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	mi := []models.GeneralMonthlyInfo{
		{
			Month:              1,
			Count:              100,
			BaseRemuneration:   10000,
			OtherRemunerations: 1000,
			Discounts:          1000,
			Remunerations:      10000,
			ItemSummary: models.ItemSummary{
				FoodAllowance:        100,
				BonusLicense:         150,
				VacationCompensation: 125,
				ChristmasBonus:       175,
				CompensatoryLicense:  120,
				HealthAllowance:      300,
				Vacation:             220,
				Others:               200,
			},
		},
		{
			Month:              2,
			Count:              200,
			BaseRemuneration:   20000,
			OtherRemunerations: 2000,
			Discounts:          1000,
			Remunerations:      21000,
			ItemSummary: models.ItemSummary{
				FoodAllowance: 100,
				Others:        200,
			},
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetGeneralMonthlyInfosFromYear(2020).Return(mi, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/geral/remuneracao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("ano")
	ctx.SetParamValues("2020")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetGeneralRemunerationFromYear(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		[
			{
				"mes": 1,
				"num_membros": 100,
				"remuneracao_base": 10000,
				"outras_remuneracoes": 1000,
				"descontos": 1000,
				"remuneracoes": 10000,
				"resumo_rubricas": {
					"auxilio_alimentacao": 100,
					"licenca_premio": 150,
					"indenizacao_de_ferias": 125,
					"ferias": 220,
					"gratificacao_natalina": 175,
					"licenca_compensatoria": 120,
					"auxilio_saude": 300,
					"outras": 200
				  }
			},
			{
				"mes": 2,
				"num_membros": 200,
				"remuneracao_base": 20000,
				"outras_remuneracoes": 2000,
				"descontos": 1000,
				"remuneracoes": 21000,
				"resumo_rubricas": {
					"auxilio_alimentacao": 100,
					"licenca_premio": 0,
					"indenizacao_de_ferias": 0,
					"ferias": 0,
					"gratificacao_natalina": 0,
					"licenca_compensatoria": 0,
					"auxilio_saude": 0,
					"outras": 200
				  }
			}
		]
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getGenerealRemunerationFromYear) testWhenDataDoesNotExist(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetGeneralMonthlyInfosFromYear(2020).Return([]models.GeneralMonthlyInfo{}, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/geral/remuneracao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("ano")
	ctx.SetParamValues("2020")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetGeneralRemunerationFromYear(ctx)

	expectedCode := http.StatusOK
	expectedJson := `[]`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getGenerealRemunerationFromYear) testWhenYearIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v2/geral/remuneracao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("ano")
	ctx.SetParamValues("2020a")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetGeneralRemunerationFromYear(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func TestGetTotalsOfAgencyYear(t *testing.T) {
	tests := getTotalsOfAgencyYear{}
	t.Run("test when data exists", tests.testWhenDataExists)
	t.Run("test when monthly info does not exist", getTotalsOfAgencyYear{}.testWhenMonthlyInfoDoesNotExist)
	t.Run("test when year is invalid", tests.testWhenYearIsInvalid)
}

type getTotalsOfAgencyYear struct{}

func (g getTotalsOfAgencyYear) testWhenDataExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agmi := agencyMonthlyInfos()
	agencies := []models.Agency{
		{
			ID:            "tjal",
			Name:          "Tribunal de Justiça do Estado de Alagoas",
			Type:          "Estadual",
			Entity:        "Tribunal",
			UF:            "AL",
			TwitterHandle: "tjaloficial",
			OmbudsmanURL:  "http://www.tjal.jus.br/ombudsman",
		},
	}
	avg := models.PerCapitaData{
		AgencyID:           "tjal",
		Year:               2020,
		BaseRemuneration:   33173.01121495333,
		OtherRemunerations: 9119.563364485992,
		Discounts:          10382.615233644861,
		Remunerations:      33173.01121495333,
	}

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(&agencies[0], nil).Times(1)
	dbMock.EXPECT().GetMonthlyInfo([]models.Agency{{ID: "tjal"}}, 2020).Return(map[string][]models.AgencyMonthlyInfo{"tjal": agmi}, nil).Times(1)
	fsMock.EXPECT().GetFile("tjal/datapackage/tjal-2020.zip").Return(agmi[0].Package, nil)
	dbMock.EXPECT().GetAveragePerCapita("tjal", 2020).Return(&avg, nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/totais/:orgao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano")
	ctx.SetParamValues("tjal", "2020")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetTotalsOfAgencyYear(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"ano": 2020,
			"orgao": {
				"id_orgao": "tjal",
				"nome": "Tribunal de Justiça do Estado de Alagoas",
				"jurisdicao": "Estadual",
				"entidade": "Tribunal",
				"uf": "AL",
				"twitter_handle": "tjaloficial",
				"ouvidoria": "http://www.tjal.jus.br/ombudsman",
				"url": "example.com/v2/orgao/tjal"
			},
			"media_por_membro": {
				"remuneracao_base": 33173.01121495333,
				"outras_remuneracoes": 9119.563364485992,
				"descontos": 10382.615233644861,
				"remuneracoes": 33173.01121495333
			  },
			"meses": [
				{
					"mes": 1,
					"outras_remuneracoes":1.9515865600000022e+06,
					"outras_remuneracoes_por_membro":9119.563364485992,
					"remuneracao_base":7.099024400000013e+06,
					"remuneracao_base_por_membro":33173.01121495333,
					"descontos": 2221879.66,
      				"descontos_por_membro": 10382.615233644861,
					"remuneracoes": 7.099024400000013e+06,
					"remuneracoes_por_membro": 33173.01121495333,
					"timestamp": {
						"seconds": 1,
						"nanos": 1
					},
					"total_membros": 214,
					"resumo_rubricas": {
						"auxilio_alimentacao": 100,
        	            "licenca_premio": 150,
						"indenizacao_de_ferias": 130,
						"ferias": 220,
						"gratificacao_natalina": 120,
						"licenca_compensatoria": 120,
						"auxilio_saude": 300,
        	            "outras": 200
					}
				}
			],
			"package": {
				"url": "https://dadosjusbr.org/download/tjal/datapackage/tjal-2020-1.zip",
				"hash": "4d7ca8986101673aea060ac1d8e5a529",
				"size": 30195
			}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getTotalsOfAgencyYear) testWhenMonthlyInfoDoesNotExist(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agency := models.Agency{
		ID:            "tjal",
		Name:          "Tribunal de Justiça do Estado de Alagoas",
		Type:          "Estadual",
		Entity:        "Tribunal",
		UF:            "AL",
		TwitterHandle: "tjaloficial",
		OmbudsmanURL:  "http://www.tjal.jus.br/ombudsman",
	}
	avg := models.PerCapitaData{
		AgencyID:           "tjal",
		Year:               2020,
		BaseRemuneration:   0,
		OtherRemunerations: 0,
		Discounts:          0,
		Remunerations:      0,
	}

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(&agency, nil).Times(1)
	dbMock.EXPECT().GetMonthlyInfo([]models.Agency{{ID: "tjal"}}, 2020).Return(nil, nil).Times(1)
	fsMock.EXPECT().GetFile("tjal/datapackage/tjal-2020.zip").Return(nil, nil).Times(1)
	dbMock.EXPECT().GetAveragePerCapita("tjal", 2020).Return(&avg, nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/totais/:orgao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano")
	ctx.SetParamValues("tjal", "2020")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetTotalsOfAgencyYear(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
				"ano": 2020,
				"orgao": {
					"id_orgao": "tjal",
					"nome": "Tribunal de Justiça do Estado de Alagoas",
					"jurisdicao": "Estadual",
					"entidade": "Tribunal",
					"uf": "AL",
					"twitter_handle": "tjaloficial",
					"ouvidoria": "http://www.tjal.jus.br/ombudsman",
					"url": "example.com/v2/orgao/tjal"
				},
				"media_por_membro": {
					"remuneracao_base": 0,
					"outras_remuneracoes": 0,
					"descontos": 0,
					"remuneracoes": 0
				}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getTotalsOfAgencyYear) testWhenYearIsInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/totais/:orgao/:ano",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao", "ano")
	ctx.SetParamValues("tjal", "2020a")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.V2GetTotalsOfAgencyYear(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro ano=2020a inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func TestGetAnnualSummary(t *testing.T) {
	tests := getAnnualSummary{}
	t.Run("Test GetAnnualSummary when data exists", tests.testWhenDataExists)
	t.Run("Test GetAnnualSummary when agency does not exist", tests.testWhenAgencyDoesNotExist)
	t.Run("Test GetAnnualSummary when GetAnnualSummary() returns error", tests.testWhenGetAnnualSummaryReturnsError)
	t.Run("Test GetAnnualSummary when agency does not have data", tests.testWhenAgencyDoesNotHaveData)
}

type getAnnualSummary struct{}

func (g getAnnualSummary) testWhenDataExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agency := models.Agency{
		ID:            "tjal",
		Name:          "Tribunal de Justiça do Estado de Alagoas",
		Type:          "Estadual",
		Entity:        "Tribunal",
		UF:            "AL",
		TwitterHandle: "tjaloficial",
		OmbudsmanURL:  "http://www.tjal.jus.br/ombudsman",
	}
	agmi := []models.AnnualSummary{
		{
			Year:                        2020,
			AverageCount:                214,
			TotalCount:                  2568,
			BaseRemuneration:            10000,
			BaseRemunerationPerCapita:   3.8940809968847354,
			OtherRemunerations:          1000,
			OtherRemunerationsPerCapita: 0.3894080996884735,
			Discounts:                   1000,
			DiscountsPerCapita:          0.3894080996884735,
			Remunerations:               10000,
			RemunerationsPerCapita:      3.8940809968847354,
			NumMonthsWithData:           12,
			Package: &models.Backup{
				URL:  "https://dadosjusbr.org/download/tjal/datapackage/tjal-2020-1.zip",
				Hash: "4d7ca8986101673aea060ac1d8e5a529",
				Size: 30195,
			},
			ItemSummary: models.ItemSummary{
				FoodAllowance:        100,
				BonusLicense:         150,
				VacationCompensation: 130,
				Vacation:             220,
				ChristmasBonus:       170,
				CompensatoryLicense:  120,
				HealthAllowance:      300,
				Others:               200,
			},
		},
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(&agency, nil).Times(1)
	dbMock.EXPECT().GetAnnualSummary("tjal").Return(agmi, nil).Times(1)
	fsMock.EXPECT().GetFile("tjal/datapackage/tjal-2020.zip").Return(agmi[0].Package, nil)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/resumo/:orgao",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao")
	ctx.SetParamValues("tjal")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetAnnualSummary(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"orgao": {
				"id_orgao": "tjal",
				"nome": "Tribunal de Justiça do Estado de Alagoas",
				"jurisdicao": "Estadual",
				"entidade": "Tribunal",
				"uf": "AL",
				"twitter_handle": "tjaloficial",
				"ouvidoria": "http://www.tjal.jus.br/ombudsman",
				"url": "example.com/v2/orgao/tjal"
			},
			"timestamp": {
				"nanos":0, 
				"seconds":0
			},
			"dados_anuais": [
				{
					"ano": 2020,
					"num_membros": 214,
					"remuneracao_base": 10000,
					"remuneracao_base_por_membro": 3.8940809968847354, 
					"remuneracao_base_por_mes": 833.3333333333334,
					"outras_remuneracoes": 1000,
					"outras_remuneracoes_por_membro": 0.3894080996884735, 
					"outras_remuneracoes_por_mes": 83.33333333333333,
					"descontos": 1000,
					"descontos_por_membro": 0.3894080996884735, 
					"descontos_por_mes": 83.33333333333333,
					"remuneracoes": 10000,
					"remuneracoes_por_membro": 3.8940809968847354,
					"remuneracoes_por_mes": 833.3333333333334,
					"meses_com_dados": 12,
					"package": {
						"url": "https://dadosjusbr.org/download/tjal/datapackage/tjal-2020-1.zip",
						"hash": "4d7ca8986101673aea060ac1d8e5a529",
						"size": 30195
					},
					"resumo_rubricas": {
						"auxilio_alimentacao": 100,
						"licenca_premio": 150,
						"indenizacao_de_ferias": 130,
						"ferias": 220,
						"gratificacao_natalina": 170,
						"licenca_compensatoria": 120,
						"auxilio_saude": 300,
						"outras": 200
					  }
				}
			]
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func (g getAnnualSummary) testWhenAgencyDoesNotExist(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(nil, fmt.Errorf("error")).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/resumo/:orgao",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao")
	ctx.SetParamValues("tjal")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetAnnualSummary(ctx)

	expectedCode := http.StatusBadRequest
	expectedJson := `"Parâmetro orgao=tjal inválido"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getAnnualSummary) testWhenGetAnnualSummaryReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agency := models.Agency{
		ID:            "tjal",
		Name:          "Tribunal de Justiça do Estado de Alagoas",
		Entity:        "Tribunal",
		UF:            "AL",
		TwitterHandle: "tjaloficial",
		Type:          "Estadual",
		URL:           "example.com/v2/orgao/tjal",
		OmbudsmanURL:  "http://www.tjal.jus.br/ombudsman",
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(&agency, nil).Times(1)
	dbMock.EXPECT().GetAnnualSummary("tjal").Return(nil, fmt.Errorf("error")).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v2/orgao/resumo/:orgao",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao")
	ctx.SetParamValues("tjal")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetAnnualSummary(ctx)

	expectedCode := http.StatusInternalServerError
	expectedJson := `"Algo deu errado ao tentar coletar os dados anuais do orgao=tjal"`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.Equal(t, expectedJson, strings.Trim(recorder.Body.String(), "\n"))
}

func (g getAnnualSummary) testWhenAgencyDoesNotHaveData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dbMock := database.NewMockInterface(mockCtrl)
	fsMock := file_storage.NewMockInterface(mockCtrl)

	agency := models.Agency{
		ID:            "tjal",
		Name:          "Tribunal de Justiça do Estado de Alagoas",
		Entity:        "Tribunal",
		UF:            "AL",
		TwitterHandle: "tjaloficial",
		Type:          "Estadual",
		URL:           "example.com/v2/orgao/tjal",
		OmbudsmanURL:  "http://www.tjal.jus.br/ombudsman",
	}
	dbMock.EXPECT().Connect().Return(nil).Times(1)
	dbMock.EXPECT().GetAgency("tjal").Return(&agency, nil).Times(1)
	dbMock.EXPECT().GetAnnualSummary("tjal").Return([]models.AnnualSummary{}, nil).Times(1)

	e := echo.New()
	request := httptest.NewRequest(
		http.MethodGet,
		"/uiapi/v1/orgao/resumo/:orgao",
		nil,
	)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	ctx.SetParamNames("orgao")
	ctx.SetParamValues("tjal")

	client, _ := storage.NewClient(dbMock, fsMock)
	handler, err := NewHandler(client, nil, nil, "us-east-1", "dadosjusbr_public", loc, []string{}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	handler.GetAnnualSummary(ctx)

	expectedCode := http.StatusOK
	expectedJson := `
		{
			"orgao": {
				"id_orgao": "tjal",
				"nome": "Tribunal de Justiça do Estado de Alagoas",
				"jurisdicao": "Estadual",
				"entidade": "Tribunal",
				"uf": "AL",
				"twitter_handle": "tjaloficial",
				"ouvidoria": "http://www.tjal.jus.br/ombudsman",
				"url": "example.com/v2/orgao/tjal"
			},
			"timestamp": {
				"nanos":0, 
				"seconds":0
			}
		}
	`

	assert.Equal(t, expectedCode, recorder.Code)
	assert.JSONEq(t, expectedJson, recorder.Body.String())
}

func agencyMonthlyInfos() []models.AgencyMonthlyInfo {
	return []models.AgencyMonthlyInfo{
		{
			AgencyID: "tjal",
			Month:    1,
			Year:     2020,
			Summary: &models.Summary{
				Count: 214,
				BaseRemuneration: models.DataSummary{
					Max:     35462.22,
					Min:     7473.09,
					Average: 33173.01121495333,
					Total:   7099024.400000013,
				},
				OtherRemunerations: models.DataSummary{
					Max:     45200.05,
					Average: 9119.563364485992,
					Total:   1951586.5600000022,
				},
				Discounts: models.DataSummary{
					Max:     23118.190000000002,
					Average: 10382.615233644861,
					Total:   2221879.66,
				},
				Remunerations: models.DataSummary{
					Max:     35462.22,
					Min:     7473.09,
					Average: 33173.01121495333,
					Total:   7099024.400000013,
				},
				IncomeHistogram: map[int]int{
					-1:    0,
					10000: 1,
					20000: 0,
					30000: 3,
					40000: 210,
					50000: 0,
				},
				ItemSummary: models.ItemSummary{
					FoodAllowance:        100,
					BonusLicense:         150,
					VacationCompensation: 130,
					Vacation:             220,
					ChristmasBonus:       120,
					CompensatoryLicense:  120,
					HealthAllowance:      300,
					Others:               200,
				},
			},
			CrawlerVersion:    "unspecified",
			CrawlerRepo:       "https://github.com/dadosjusbr/coletor-cnj",
			CrawlingTimestamp: timestamppb.New(time.Unix(1, 1)),
			Package: &models.Backup{
				URL:  "https://dadosjusbr.org/download/tjal/datapackage/tjal-2020-1.zip",
				Hash: "4d7ca8986101673aea060ac1d8e5a529",
				Size: 30195,
			},
			Meta: &models.Meta{
				OpenFormat:       false,
				Access:           "NECESSITA_SIMULACAO_USUARIO",
				Extension:        "XLS",
				StrictlyTabular:  true,
				ConsistentFormat: true,
				HaveEnrollment:   false,
				ThereIsACapacity: false,
				HasPosition:      false,
				BaseRevenue:      "DETALHADO",
				OtherRecipes:     "DETALHADO",
				Expenditure:      "DETALHADO",
			},
			Score: &models.Score{
				Score:             0.5,
				CompletenessScore: 0.5,
				EasinessScore:     0.5,
			},
			ProcInfo: &coleta.ProcInfo{},
		},
	}
}
