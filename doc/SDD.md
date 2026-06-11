# SDD - GitHub Action de Cache S3 (Linux-only)

## 1. Visão Geral

Este projeto implementa uma GitHub Action para cache de artefatos de build em Amazon S3.

A action suporta três operações:
- `put`: compacta artefatos locais em `.zip` e envia para o S3.
- `get`: verifica existência da chave no S3, baixa o `.zip` e extrai no workspace.
- `delete`: remove o objeto de cache no S3.

Objetivo principal:
- reduzir tempo de pipeline reutilizando dependências/artefatos entre execuções;
- centralizar cache em bucket S3 gerenciado pelo time.

Escopo de plataforma:
- suporte somente a runner Linux.

## 2. Requisitos Funcionais

### RF-01: Entrada da operação
A action deve receber input obrigatório `action` com valores válidos:
- `put`
- `get`
- `delete`

Qualquer valor fora dessa lista deve encerrar com erro.

### RF-02: Inputs obrigatórios de armazenamento
A action deve receber:
- `aws-region` (obrigatório)
- `bucket` (obrigatório)
- `key` (obrigatório)

A chave efetiva usada no S3 deve ser `key + ".zip"`.

### RF-03: Inputs opcionais
A action deve suportar:
- `artifacts` (opcional; necessário quando `action=put`)
- `s3-class` (opcional; default `STANDARD`)

### RF-04: Operação PUT
Quando `action=put`:
1. Ler `artifacts` como lista de padrões (linhas separadas por newline).
2. Expandir padrões glob com `filepath.Glob`.
3. Percorrer recursivamente cada match com `filepath.Walk`.
4. Criar arquivo zip local `key.zip`.
5. Adicionar entradas de arquivo e diretório no zip mantendo `header.Name = path`.
6. Fazer upload no S3 com `StorageClass = s3-class`.
7. Remover `key.zip` local ao final.

Se não houver padrão válido em `artifacts`, deve falhar.

### RF-05: Operação GET
Quando `action=get`:
1. Executar `HeadObject` para verificar se o cache existe.
2. Se existir:
   - baixar `key.zip` do S3;
   - extrair conteúdo no filesystem local;
   - remover `key.zip` local;
   - publicar output `CACHE_HIT=true` em `GITHUB_OUTPUT`.
3. Se não existir:
   - apenas registrar log de cache miss;
   - não publicar `CACHE_HIT=true`.

### RF-06: Operação DELETE
Quando `action=delete`:
- excluir objeto `key.zip` no bucket S3.

### RF-07: Output da Action
A action deve expor output:
- `cache-hit`: mapeado para `${{ steps.cache.outputs.CACHE_HIT }}`.

## 3. Requisitos Não Funcionais

- Linguagem: Go.
- Dependência AWS: `github.com/aws/aws-sdk-go` (v1).
- Runner alvo: Linux.
- Distribuição da action: `composite` com execução de binário Linux em `dist/linux`.
- Logging simples via `log.Print`/`log.Printf`.

## 4. Arquitetura

## 4.1 Componentes

- `action.yml`
  - Define metadata da action, inputs, output e step único Linux.

- `entrypoint.sh`
  - Executa binário Linux em `$GITHUB_ACTION_PATH/dist/linux`.

- `main.go`
  - Orquestra fluxo por operação (`put`, `get`, `delete`).
  - Monta objeto `Action` a partir de variáveis de ambiente.

- `types.go`
  - Constantes de operação e struct de entrada.

- `archive.go`
  - Implementa compactação (`Zip`) e extração (`Unzip`).

- `s3.go`
  - Implementa integrações AWS S3:
    - `PutObject`
    - `GetObject`
    - `DeleteObject`
    - `ObjectExists`

## 4.2 Fluxo de Execução

1. GitHub Action injeta env vars dos inputs.
2. `entrypoint.sh` executa `dist/linux`.
3. Binário Go lê:
   - `ACTION`
   - `BUCKET`
   - `S3_CLASS`
   - `KEY`
   - `ARTIFACTS`
4. `main.go` escolhe fluxo por `ACTION`.
5. Operações de zip/s3 são executadas.
6. Em cache hit de `get`, escreve `CACHE_HIT=true` em `GITHUB_OUTPUT`.

## 5. Contrato da Action

## 5.1 Inputs (action.yml)

- `action` (required)
- `aws-region` (required)
- `bucket` (required)
- `key` (required)
- `artifacts` (optional)
- `s3-class` (optional, default `STANDARD`)

## 5.2 Variáveis de ambiente consumidas no runtime

- `ACTION`
- `AWS_REGION`
- `BUCKET`
- `S3_CLASS`
- `KEY`
- `ARTIFACTS`
- `GITHUB_OUTPUT`

## 5.3 Output

- `cache-hit`

## 5.4 Integração com GitHub Workflows

Status nesta SDD: funcional e obrigatória.

Definição de obrigatoriedade aprovada:
- obrigatório manter exatamente dois workflows neste repositório para validação e uso de referência;
- obrigatório existir workflow consumidor nos repositórios que utilizam a action.

Política de workflows deste repositório (Linux-only):
1. `push` em `feature/**` para validação contínua, atualização de `dist/linux` e publicação automática do resultado na própria branch.
2. `pull_request` para `main` para validação de integração antes do merge.

Workflow de referência para `push` em `feature/**`:

```yaml
name: Action Build CI

on:
  push:
    branches:
      - 'feature/**'

jobs:
  build-action:
    runs-on: itau-linux-k8s-on-demand-medium
    steps:
      - name: Checkout
        uses: actions/checkout@v2
        with:
          fetch-depth: 0

      - name: Setup golang
        uses: actions/setup-go@v2
        with:
          go-version: ^1.15.5

      - name: Build binary
        run: |
          env GOOS=linux GOARCH=amd64 go build -o dist/linux
          chmod +x dist/linux
```

Workflow de referência para `pull_request` em `main`:

```yaml
name: PR Validation

on:
  pull_request:
    types:
      - opened
      - synchronize
      - reopened
    branches:
      - 'main'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup golang
        uses: actions/setup-go@v5
        with:
          go-version: '1.16'

      - name: Build linux binary
        run: |
          mkdir -p dist
          env GOOS=linux GOARCH=amd64 go build -o dist/linux
          chmod +x dist/linux
```

Requisitos de implementação do workflow:
1. Manter somente os dois gatilhos aprovados: `push` em `feature/**` e `pull_request` para `main`.
2. Buildar somente binário Linux (`dist/linux`).
3. Não incluir branchs Windows/macOS nem workflows extras de stale/release neste repositório.
4. O workflow de feature serve como validação contínua, geração do binário Linux e commit/push automático do artefato para a própria branch.
5. O passo de publicação deve permanecer explícito no workflow de feature para manter o binário versionado no repositório.

## 6. Decisões de Design

1. Linux-only
- simplifica manutenção e evita ramos de execução por OS.
- remove necessidade de `entrypoint.bat` e step Windows.

2. Action do tipo `composite` mantida
- evita mudança de arquitetura para Docker Action.
- preserva forma atual de integração em workflows existentes.

3. Clean-up aplicado sem mudança funcional de alto impacto
- remoção de função não usada (`setEnv`).
- correção de tratamento de erro em `filepath.Walk` e `GetObject`.
- remoção de `defer` dentro de loop no unzip para reduzir risco de exaustão de descritores.
- padronização de typo (`writter` -> `writer`).

## 7. Estrutura Esperada do Repositório

- `action.yml`
- `entrypoint.sh`
- `main.go`
- `types.go`
- `archive.go`
- `s3.go`
- `go.mod`
- `go.sum`
- `dist/linux` (binário executável distribuído com a action)
- `docs/SDD.md`

## 8. Regras de Erro e Observabilidade

- Falhas de validação e execução devem encerrar com código de erro (`log.Fatal`).
- Mensagens de sucesso esperadas:
  - upload: `Cache saved successfully`
  - download: `Cache downloaded successfully, containing <bytes> bytes`
  - delete: `Cache purged successfully`
  - hit: `Cache hit for the following key: <key.zip>`
  - miss: `No caches found for the following key: <key.zip>`

## 9. Limitações Conhecidas (mantidas para compatibilidade)

- O nome dos arquivos no zip usa o path retornado no walk (`header.Name = path`).
- A extração usa diretamente `file.Name` do zip.
- Não há criptografia customizada nem versionamento interno de cache.

## 10. Critérios de Aceite

1. `action=put` com `artifacts` válido gera `key.zip`, envia para S3 e remove zip local.
2. `action=get` com objeto existente baixa, extrai e publica `cache-hit=true`.
3. `action=get` sem objeto existente não falha e não publica `cache-hit=true`.
4. `action=delete` remove o objeto no bucket.
5. Execução em runner Linux funciona sem qualquer branch para Windows.
6. Existem exatamente dois workflows neste repositório: validação de `push` em `feature/**` e validação de `pull_request` para `main`.
7. Os workflows do repositório buildam somente `dist/linux`.
8. O workflow de `push` em `feature/**` mantém commit/push automático do binário gerado.
9. Workflows consumidores podem usar `cache-hit` para pular etapas de build quando houver cache hit.

## 11. Guia de Implementação para Outra AI

1. Criar ação `composite` com um único step shell Linux.
2. Ler inputs e mapear para env vars conforme contrato.
3. Implementar runtime Go com switch por operação.
4. Implementar zip/unzip e S3 SDK conforme RFs.
5. Garantir output `CACHE_HIT=true` apenas em cache hit real.
6. Distribuir binário Linux em `dist/linux`.
7. Criar e manter somente dois workflows no repositório (`feature/**` e PR para `main`).
8. Manter o passo de publicação automática no workflow de `feature/**`.
9. Validar fluxos `put/get/delete` em pipeline de teste de repositório consumidor.
