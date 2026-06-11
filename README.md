# up-action-cache-s3

GitHub Action para cache de artefatos de build em Amazon S3. Suporta operações de `put`, `get` e `delete`. Linux-only.

## Inputs

| Input | Obrigatório | Default | Descrição |
|---|---|---|---|
| `action` | sim | — | Operação: `put`, `get` ou `delete` |
| `aws-region` | sim | — | Região AWS |
| `bucket` | sim | — | Nome do bucket S3 |
| `key` | sim | — | Chave de cache (a chave efetiva no S3 será `key.zip`) |
| `artifacts` | não* | — | Padrões glob separados por newline (obrigatório para `put`) |
| `s3-class` | não | `STANDARD` | Storage class do S3 |

## Outputs

| Output | Descrição |
|---|---|
| `cache-hit` | `true` quando o cache foi encontrado e extraído com sucesso |

## Uso

```yaml
- name: Restore cache
  id: cache
  uses: <org>/up-action-cache-s3@main
  with:
    action: get
    aws-region: us-east-1
    bucket: my-cache-bucket
    key: my-project-${{ hashFiles('go.sum') }}

- name: Install dependencies
  if: steps.cache.outputs.cache-hit != 'true'
  run: go mod download

- name: Save cache
  if: steps.cache.outputs.cache-hit != 'true'
  uses: <org>/up-action-cache-s3@main
  with:
    action: put
    aws-region: us-east-1
    bucket: my-cache-bucket
    key: my-project-${{ hashFiles('go.sum') }}
    artifacts: |
      vendor
      ~/.cache/go

- name: Delete cache
  uses: <org>/up-action-cache-s3@main
  with:
    action: delete
    aws-region: us-east-1
    bucket: my-cache-bucket
    key: my-project-${{ hashFiles('go.sum') }}
```

## Desenvolvimento

**Compilar o binário Linux (obrigatório antes de commitar):**

```bash
env GOOS=linux GOARCH=amd64 go build -o dist/linux
chmod +x dist/linux
```

O binário `dist/linux` deve ser versionado no repositório. O workflow de `feature/**` faz esse commit automaticamente após cada push.

**Atualizar dependências:**

```bash
go mod tidy
```

## Workflows

| Workflow | Gatilho | Descrição |
|---|---|---|
| `feature.yml` | Push em `feature/**` | Build + commit automático de `dist/linux` |
| `pr.yml` | PR para `main` | Validação de build antes do merge |
