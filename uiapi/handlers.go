package uiapi

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dadosjusbr/storage"
	strModels "github.com/dadosjusbr/storage/models"
	"github.com/gocarina/gocsv"
	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"gorm.io/gorm"
)

type handler struct {
	client           storage.Client
	db               *postgresDB
	sess             *awsSession
	s3Bucket         string
	loc              *time.Location
	envOmittedFields []string
	searchLimit      int
	downloadLimit    int
}

func NewHandler(client storage.Client, conn *gorm.DB, newrelic *newrelic.Application, awsRegion string, s3Bucket string, loc *time.Location, envOmittedFields []string, searchLimit, downloadLimit int) (*handler, error) {
	db := &postgresDB{
		conn:     conn,
		newrelic: newrelic,
	}
	sess, err := newAwsSession(awsRegion)
	if err != nil {
		return nil, err
	}
	return &handler{
		db:               db,
		sess:             sess,
		s3Bucket:         s3Bucket,
		client:           client,
		loc:              loc,
		envOmittedFields: envOmittedFields,
		searchLimit:      searchLimit,
		downloadLimit:    downloadLimit,
	}, nil
}

func (h handler) GetSummaryOfAgency(c echo.Context) error {
	year, err := strconv.Atoi(c.Param("ano"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d inválido", year))
	}
	month, err := strconv.Atoi(c.Param("mes"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro mês=%d", month))
	}
	agencyName := c.Param("orgao")
	agencyMonthlyInfo, agency, err := h.client.GetOMA(month, year, agencyName)
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d, mês=%d ou nome do orgão=%s são inválidos", year, month, agencyName))
	}
	agencySummary := agencySummary{
		FullName:   agency.Name,
		TotalWage:  agencyMonthlyInfo.Summary.BaseRemuneration.Total,
		MaxWage:    agencyMonthlyInfo.Summary.BaseRemuneration.Max,
		TotalPerks: agencyMonthlyInfo.Summary.OtherRemunerations.Total,
		MaxPerk:    agencyMonthlyInfo.Summary.OtherRemunerations.Max,
		TotalRemuneration: agencyMonthlyInfo.Summary.BaseRemuneration.Total +
			agencyMonthlyInfo.Summary.OtherRemunerations.Total,
		TotalMembers: agencyMonthlyInfo.Summary.Count,
		CrawlingTime: agencyMonthlyInfo.CrawlingTimestamp,
		HasNext:      time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).In(h.loc).Before(time.Now().AddDate(0, 1, 0)),
		HasPrevious:  time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).In(h.loc).After(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).In(h.loc)),
	}
	return c.JSON(http.StatusOK, agencySummary)
}

func (h handler) GetSalaryOfAgencyMonthYear(c echo.Context) error {
	month, err := strconv.Atoi(c.Param("mes"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro mês=%d", month))
	}
	year, err := strconv.Atoi(c.Param("ano"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d", year))
	}
	agencyName := strings.ToLower(c.Param("orgao"))
	agencyMonthlyInfo, _, err := h.client.GetOMA(month, year, agencyName)
	if err != nil {
		log.Printf("[salary agency month year] error getting data for second screen(mes:%d ano:%d, orgao:%s):%q", month, year, agencyName, err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d, mês=%d ou nome do orgão=%s são inválidos", year, month, agencyName))
	}
	if agencyMonthlyInfo.ProcInfo.String() != "" {
		var newEnv = agencyMonthlyInfo.ProcInfo.Env
		for _, omittedField := range h.envOmittedFields {
			for i, field := range newEnv {
				if strings.Contains(field, omittedField) {
					newEnv[i] = omittedField + "= ##omitida##"
					break
				}
			}
		}
		agencyMonthlyInfo.ProcInfo.Env = newEnv
		return c.JSON(http.StatusPartialContent, procInfoResult{
			ProcInfo:          agencyMonthlyInfo.ProcInfo,
			CrawlingTimestamp: agencyMonthlyInfo.CrawlingTimestamp,
		})
	}
	return c.JSON(http.StatusOK, dataForChartAtAgencyScreen{
		Members:     agencyMonthlyInfo.Summary.IncomeHistogram,
		MaxSalary:   agencyMonthlyInfo.Summary.BaseRemuneration.Max,
		PackageURL:  agencyMonthlyInfo.Package.URL,
		PackageHash: agencyMonthlyInfo.Package.Hash,
		PackageSize: agencyMonthlyInfo.Package.Size,
	})
}

func (h handler) GetTotalsOfAgencyYear(c echo.Context) error {
	year, err := strconv.Atoi(c.Param("ano"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d inválido", year))
	}
	aID := c.Param("orgao")
	agenciesMonthlyInfo, err := h.client.Db.GetMonthlyInfo([]strModels.Agency{{ID: aID}}, year)
	if err != nil {
		log.Printf("[totals of agency year] error getting data for first screen(ano:%d, estado:%s):%q", year, aID, err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d ou orgao=%s inválidos", year, aID))
	}
	var monthTotalsOfYear []monthTotals
	agency, err := h.client.Db.GetAgency(aID)
	if err != nil {
		log.Printf("[totals of agency year] error getting data for first screen(estado:%s):%q", aID, err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro orgao=%s inválido", aID))
	}
	host := c.Request().Host
	agency.URL = fmt.Sprintf("%s/v1/orgao/%s", host, agency.ID)
	for _, agencyMonthlyInfo := range agenciesMonthlyInfo[aID] {
		if agencyMonthlyInfo.Summary != nil && agencyMonthlyInfo.Summary.BaseRemuneration.Total+agencyMonthlyInfo.Summary.OtherRemunerations.Total > 0 {
			monthTotals := monthTotals{Month: agencyMonthlyInfo.Month,
				BaseRemuneration:   agencyMonthlyInfo.Summary.BaseRemuneration.Total,
				OtherRemunerations: agencyMonthlyInfo.Summary.OtherRemunerations.Total,
				CrawlingTimestamp:  agencyMonthlyInfo.CrawlingTimestamp,
				TotalMembers:       agencyMonthlyInfo.Summary.Count,
			}
			monthTotalsOfYear = append(monthTotalsOfYear, monthTotals)

			// The status 4 is a report from crawlers that data is unavailable or malformed. By removing them from the API results, we make sure they are displayed as if there is no data.
		} else if agencyMonthlyInfo.ProcInfo.String() != "" && agencyMonthlyInfo.ProcInfo.Status != 4 {
			monthTotals := monthTotals{Month: agencyMonthlyInfo.Month,
				BaseRemuneration:   0,
				OtherRemunerations: 0,
				CrawlingTimestamp:  agencyMonthlyInfo.CrawlingTimestamp,
				Error:              &procError{Stdout: agencyMonthlyInfo.ProcInfo.Stdout, Stderr: agencyMonthlyInfo.ProcInfo.Stderr},
			}
			monthTotalsOfYear = append(monthTotalsOfYear, monthTotals)
		}
	}
	sort.Slice(monthTotalsOfYear, func(i, j int) bool {
		return monthTotalsOfYear[i].Month < monthTotalsOfYear[j].Month
	})
	destKey := fmt.Sprintf("%s/datapackage/%s-%d.zip", aID, aID, year)
	bkp, _ := h.client.Cloud.GetFile(destKey)
	var pkg *strModels.Package
	if bkp != nil {
		pkg = &strModels.Package{
			AgencyID: &aID,
			Year:     &year,
			Package:  *bkp,
		}
	}

	agencyTotalsYear := agencyTotalsYear{Year: year, Agency: agency, MonthTotals: monthTotalsOfYear, AgencyFullName: agency.Name, SummaryPackage: pkg}
	return c.JSON(http.StatusOK, agencyTotalsYear)
}

func (h handler) GetBasicInfoOfType(c echo.Context) error {
	yearOfConsult := time.Now().Year()
	groupName := c.Param("grupo")
	var agencies []strModels.Agency
	var err error
	var estadual bool
	var exists bool
	jurisdicao := map[string]string{
		"justica-eleitoral":    "Eleitoral",
		"ministerios-publicos": "Ministério",
		"justica-estadual":     "Estadual",
		"justica-do-trabalho":  "Trabalho",
		"justica-federal":      "Federal",
		"justica-militar":      "Militar",
		"justica-superior":     "Superior",
		"conselhos-de-justica": "Conselho",
	}

	// Adaptando as URLs do site com o banco de dados
	// Primeiro consultamos entre as chaves do mapa.
	if jurisdicao[groupName] != "" {
		groupName = jurisdicao[groupName]
	} else {
		// Caso não encontremos entre as chaves, verificamos entre os valores do mapa.
		// Isso pois, até a consolidação ser finalizada, o front consulta a api com /Eleitoral, /Trabalho, etc.
		for _, value := range jurisdicao {
			if groupName == value {
				exists = true
				break
			}
		}
		// Se a jurisdição não existir no mapa, verificamos se trata-se de um estado
		if !exists {
			values := map[string]struct{}{"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {}, "ES": {}, "GO": {}, "MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {}, "PB": {}, "PR": {}, "PE": {}, "PI": {}, "RJ": {}, "RN": {}, "RS": {}, "RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {}}
			if _, estadual = values[groupName]; estadual {
				exists = true
			}
		}
		// Se o parâmetro dado não for encontrado de forma alguma, retornamos um NOT FOUND (404)
		if !exists {
			return c.JSON(http.StatusNotFound, fmt.Sprintf("Grupo não encontrado: %s.", groupName))
		}
	}

	if estadual {
		agencies, err = h.client.GetStateAgencies(groupName)
	} else {
		agencies, err = h.client.GetOPJ(groupName)
	}
	if err != nil {
		// That happens when there is no information on that year.
		log.Printf("[basic info type] first error getting data for first screen(ano:%d, grupo:%s). Going to try again with last year:%q", yearOfConsult, groupName, err)
		yearOfConsult = yearOfConsult - 1

		if estadual {
			agencies, err = h.client.GetStateAgencies(groupName)
		} else {
			agencies, err = h.client.GetOPJ(groupName)
		}
		if err != nil {
			log.Printf("[basic info type] error getting data for first screen(ano:%d, grupo:%s):%q", yearOfConsult, groupName, err)
			return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetros ano=%d ou grupo=%s são inválidos", yearOfConsult, groupName))
		}
	}
	var agenciesBasic []agencyBasic
	for k := range agencies {
		agenciesBasic = append(agenciesBasic, agencyBasic{Name: agencies[k].ID, FullName: agencies[k].Name, AgencyCategory: agencies[k].Entity})
	}
	state := state{Name: c.Param("grupo"), ShortName: "", FlagURL: "", Agency: agenciesBasic}
	return c.JSON(http.StatusOK, state)
}

func (h handler) GetGeneralRemunerationFromYear(c echo.Context) error {
	year, err := strconv.Atoi(c.Param("ano"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro ano=%d inválido", year))
	}
	data, err := h.client.Db.GetGeneralMonthlyInfosFromYear(year)
	if err != nil {
		fmt.Println("Error searching for monthly info from year: %w", err)
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error buscando dados"))
	}
	return c.JSON(http.StatusOK, data)
}

func (h handler) GeneralSummaryHandler(c echo.Context) error {
	agencies, err := h.client.GetAgenciesCount()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Erro ao contar orgãos: %q", err))
	}
	collections, err := h.client.GetNumberOfMonthsCollected()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Erro ao contar registros: %q", err))
	}
	fmonth, fyear, err := h.client.Db.GetFirstDateWithMonthlyInfo()
	if err != nil {
		log.Printf("Error buscando dados - GetFirstDateWithRemunerationRecords: %q", err)
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Erro buscando primeiro registro de remuneração: %q", err))
	}
	fdate := time.Date(fyear, time.Month(fmonth), 2, 0, 0, 0, 0, time.UTC).In(h.loc)
	lmonth, lyear, err := h.client.GetLastDateWithMonthlyInfo()
	if err != nil {
		log.Printf("Error buscando dados - GetLastDateWithRemunerationRecords: %q", err)
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Erro buscando último registro de remuneração: %q", err))
	}
	ldate := time.Date(lyear, time.Month(lmonth), 2, 0, 0, 0, 0, time.UTC).In(h.loc)
	remuValue, err := h.client.Db.GetGeneralMonthlyInfo()
	if err != nil {
		log.Printf("Error buscando dados - GetGeneralRemunerationValue: %q", err)
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Erro buscando valor total de remuneração: %q", err))
	}
	return c.JSON(http.StatusOK, generalTotals{
		AgencyAmount:             int(agencies),
		MonthlyTotalsAmount:      int(collections),
		StartDate:                fdate,
		EndDate:                  ldate,
		RemunerationRecordsCount: int(collections),
		GeneralRemunerationValue: remuValue,
	})
}

func (h handler) SearchByUrl(c echo.Context) error {
	//Pegando os query params
	years := c.QueryParam("anos")
	months := c.QueryParam("meses")
	agencies := c.QueryParam("orgaos")
	categories := c.QueryParam("categorias")
	types := c.QueryParam("tipos")
	//Criando os filtros a partir dos query params e validando eles
	searchParams, err := newSearchParams(years, months, agencies, categories, types)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	var category string
	if searchParams != nil {
		category = searchParams.Category
	}
	// Pegando os resultados da pesquisa a partir dos filtros;
	results, err := h.db.filter(h.db.remunerationQuery(searchParams), h.db.arguments(searchParams))
	if err != nil {
		log.Printf("Error querying BD (searchParams or counter):%q", err)
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	remunerations, numRows, err := h.getSearchResults(h.searchLimit, category, results)
	if err != nil {
		log.Printf("Error getting search results: %q", err)
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	response := searchResponse{
		DownloadAvailable:  numRows > 0 && numRows <= h.downloadLimit,
		NumRowsIfAvailable: numRows,
		DownloadLimit:      h.downloadLimit,
		SearchLimit:        h.searchLimit,
		Results:            remunerations, // retornando os SearchLimit primeiros elementos.
	}
	return c.JSON(http.StatusOK, response)
}

func (h handler) DownloadByUrl(c echo.Context) error {
	//Pegando os query params
	years := c.QueryParam("anos")
	months := c.QueryParam("meses")
	agencies := c.QueryParam("orgaos")
	categories := c.QueryParam("categorias")
	types := c.QueryParam("tipos")

	//Criando os filtros a partir dos query params e validando eles
	searchParams, err := newSearchParams(years, months, agencies, categories, types)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	results, err := h.db.filter(h.db.remunerationQuery(searchParams), h.db.arguments(searchParams))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	searchResults, _, err := h.getSearchResults(h.downloadLimit, searchParams.Category, results)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=dadosjusbr-remuneracoes.csv")
	c.Response().Header().Set("Content-Type", c.Response().Header().Get("Content-Type"))
	err = gocsv.Marshal(searchResults, c.Response().Writer)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Errorf("erro tentando fazer download do csv: %q", err))
	}
	return nil
}

func (h handler) GetAnnualSummary(c echo.Context) error {
	agencyName := c.Param("orgao")
	strAgency, err := h.client.Db.GetAgency(agencyName)
	if err != nil {
		log.Printf("error getting agency '%s' :%q", agencyName, err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("Parâmetro orgao=%s inválido", agencyName))
	}
	host := c.Request().Host
	strAgency.URL = fmt.Sprintf("%s/v1/orgao/%s", host, strAgency.ID)
	summaries, err := h.client.GetAnnualSummary(agencyName)
	if err != nil {
		log.Printf("error getting annual data of '%s' :%q", agencyName, err)
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Algo deu errado ao tentar coletar os dados anuais do orgao=%s", agencyName))
	}
	var annualData []annualSummaryData
	for _, s := range summaries {
		annualData = append(annualData, annualSummaryData{
			Year:               s.Year,
			Count:              s.Count,
			BaseRemuneration:   s.BaseRemuneration,
			OtherRemunerations: s.OtherRemunerations,
			Package:            s.Package,
		})
	}
	var collect []collecting
	for _, c := range strAgency.Collecting {
		collect = append(collect, collecting{
			Timestamp:   c.Timestamp,
			Description: c.Description,
		})
	}
	annualSum := annualSummary{
		Agency: &agency{
			ID:            strAgency.ID,
			Name:          strAgency.Name,
			URL:           strAgency.URL,
			Type:          strAgency.Type,
			Entity:        strAgency.Entity,
			UF:            strAgency.UF,
			Collecting:    collect,
			TwitterHandle: strAgency.TwitterHandle,
			OmbudsmanURL:  strAgency.OmbudsmanURL,
		},
		Data: annualData,
	}
	return c.JSON(http.StatusOK, annualSum)
}

func (h handler) getSearchResults(limit int, category string, results []searchDetails) ([]searchResult, int, error) {
	searchResults := []searchResult{}
	numRows := 0
	if len(results) == 0 {
		return searchResults, numRows, nil
	} else {
		// A razão para essa ordenação é que quando o usuário escolhe diversos órgãos
		// provavelmente ele prefere ver dados de todos eles. Dessa forma, aumentamos
		// as chances do preview limitado retornar dados de diversos órgãos.
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Ano < results[j].Ano || results[i].Mes < results[j].Mes
		})
		searchResults, numRows, err := h.sess.getRemunerationsFromS3(limit, h.downloadLimit, category, h.s3Bucket, results)
		if err != nil {
			return nil, numRows, fmt.Errorf("failed to get remunerations from s3 %q", err)
		}
		return searchResults, numRows, nil
	}
}
