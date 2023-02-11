package papi

type backup struct {
	URL  string `json:"url,omitempty"`
	Hash string `json:"hash,omitempty"`
	Size int64  `json:"size,omitempty"`
}

type dataSummary struct {
	Max     float64 `json:"max,omitempty"`
	Min     float64 `json:"min,omitempty"`
	Average float64 `json:"media,omitempty"`
	Total   float64 `json:"total,omitempty"`
}

type summary struct {
	Count              int         `json:"quantidade,omitempty"`
	BaseRemuneration   dataSummary `json:"remuneracao_base,omitempty"`
	OtherRemunerations dataSummary `json:"outras_remuneracoes,omitempty"`
}

type summaries struct {
	MemberActive summary `json:"membros_ativos,omitempty"`
}

type metadata struct {
	OpenFormat       bool   `json:"formato_aberto"`
	Access           string `json:"acesso,omitempty"`
	Extension        string `json:"extensao,omitempty"`
	StrictlyTabular  bool   `json:"dados_estritamente_tabulares"`
	ConsistentFormat bool   `json:"manteve_consistencia_no_formato"`
	HasEnrollment    bool   `json:"tem_matricula"`
	HasCapacity      bool   `json:"tem_lotacao"`
	HasPosition      bool   `json:"tem_cargo"`
	BaseRevenue      string `json:"remuneracao_basica,omitempty"`
	OtherRecipes     string `json:"outras_receitas,omitempty"`
	Expenditure      string `json:"despesas,omitempty"`
}

type score struct {
	Score             float64 `json:"indice_transparencia"`
	CompletenessScore float64 `json:"indice_completude"`
	EasinessScore     float64 `json:"indice_facilidade"`
}

type collect struct {
	Duration       float64 `json:"duracao_segundos,omitempty"`
	CrawlerRepo    string  `json:"repositorio_coletor,omitempty"`
	CrawlerVersion string  `json:"versao_coletor,omitempty"`
	ParserRepo     string  `json:"repositorio_parser,omitempty"`
	ParserVersion  string  `json:"versao_parser,omitempty"`
}

type miError struct {
	ErrorMessage string `json:"err_msg,omitempty"`
	Status       int32  `json:"status,omitempty"`
	Cmd          string `json:"cmd,omitempty"`
}

type summaryzedMI struct {
	AgencyID string     `json:"id_orgao,omitempty"`
	Month    int        `json:"mes,omitempty"`
	Year     int        `json:"ano,omitempty"`
	Summary  *summaries `json:"sumarios,omitempty"`
	Package  *backup    `json:"pacote_de_dados,omitempty"`
	Metadata *metadata  `json:"metadados,omitempty"`
	Score    *score     `json:"indice_transparencia,omitempty"`
	Collect  *collect   `json:"dados_coleta,omitempty"`
	Error    *miError   `json:"error,omitempty"`
}

type annualSummary struct {
	AgencyID           string  `json:"id_orgao,omitempty"`
	Year               int     `json:"ano,omitempty"`
	Count              int     `json:"num_membros,omitempty"`
	BaseRemuneration   float64 `json:"remuneracao_base,omitempty"`
	OtherRemunerations float64 `json:"outras_remuneracoes,omitempty"`
}
