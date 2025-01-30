**Apresentação**

Olá. Você baixou um conjunto de dados do DadosJusBr, projeto da Transparência Brasil que obtém, padroniza e divulga contracheques do Judiciário e do Ministério Público. Caso seja a sua primeira vez por aqui, recomendamos a leitura do nosso tutorial em https://dadosjusbr.org/tutoriais. 
Antes de seguir com as análises, leve em consideração que:

- Disponibilizamos dados oficiais, coletados do Painel de Remuneração do Conselho Nacional de Justiça(CNJ) e dos portais de transparência de cada Ministério Público. Caso se depare com algum valor exorbitante, recomendamos que verifique junto às fontes originárias.

- Alguns órgãos deixam de informar dados em um ou mais meses ao longo do ano. Leve isso em consideração nas análises, em especial as de cunho comparativo.

- Metade dos órgãos do Ministério Público não permitem a coleta automatizada por meio de robôs. Esses dados são obtidos manualmente, a cada seis meses, resultando em um hiato em comparação com os demais órgãos.

- Alguns órgãos apresentam números, e não textos, nos descritivos de rubricas. Esses casos já foram levados pelo projeto aos órgãos competentes. Não se trata de erro na coleta.

- Para o Ministério Público, coletamos contracheques das planilhas “Remuneração de Membros Ativos” e “Verbas Indenizatórias e Outras Remunerações Temporárias”. Entretanto, não coletamos as planilhas “Verbas referentes a exercícios anteriores”. Considere isso, especialmente nos comparativos com o Judiciário.

- Lembre-se de transformar os descontos (lançamentos de débitos) em valores negativos.

Ficamos à disposição para sanar dúvidas através do e-mail: contato@dadosjusbr.org.

**Descrição do Pacote de Dados**

Você receberá quatro arquivos:

1. Coleta: Informações técnicas do processo de coleta dos dados de cada órgão.
2. Contracheque: Identificação nominal dos membros e valor total recebido em salário base, benefícios, descontos, bem como a remuneração líquida naquele mês. 
3. Remuneração: Remunerações e descontos agregados por órgão e por rubrica (cada nomenclatura de lançamento), separados por mês e ano.
4. Metadados: Documentação sobre a completude e a facilidade do acesso aos conjuntos de dados, que resulta no nosso índice de transparência.

O padrão Frictionless Data foi adotado para garantir que os dados tabulares utilizados no projeto sejam organizados e fáceis de trabalhar. Ele funciona como uma "etiqueta explicativa" dos dados, descrevendo informações importantes, como o que cada coluna representa, quais tipos de valores são esperados e como os dados estão estruturados.
Para o usuário, isso significa que os dados estão prontos para uso, bem documentados e mais simples de integrar com outras ferramentas ou projetos.
O pacote está licenciado sob a CC-BY-4.0, permitindo o uso e redistribuição com atribuição.
Para mais informações, acesse: https://dadosjusbr.org/sobre.