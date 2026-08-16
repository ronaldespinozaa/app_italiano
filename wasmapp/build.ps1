# Compila el motor mc a WebAssembly y copia el runtime glue (wasm_exec.js)
# junto al binario, en wasmapp/dist/.
#
# Uso (desde cualquier carpeta):
#   pwsh italian-club-app/wasmapp/build.ps1
#
# Salida: wasmapp/dist/app.wasm y wasmapp/dist/wasm_exec.js

$ErrorActionPreference = "Stop"

Push-Location $PSScriptRoot
try {
    $outDir = Join-Path $PSScriptRoot "dist"
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null

    Write-Host "Compilando app.wasm (GOOS=js GOARCH=wasm)..."
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    go build -o (Join-Path $outDir "app.wasm") .
    Remove-Item Env:\GOOS
    Remove-Item Env:\GOARCH

    # La ubicación de wasm_exec.js cambió entre versiones de Go
    # (misc/wasm/ en versiones viejas, lib/wasm/ desde Go 1.24+).
    $goroot = go env GOROOT
    $candidates = @(
        (Join-Path $goroot "lib\wasm\wasm_exec.js"),
        (Join-Path $goroot "misc\wasm\wasm_exec.js")
    )
    $src = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
    if (-not $src) {
        throw "No se encontró wasm_exec.js en $goroot (revisá lib\wasm o misc\wasm segun tu version de Go)"
    }
    Copy-Item $src (Join-Path $outDir "wasm_exec.js") -Force

    # prototype/ es lo que realmente se sirve/deploya (sin paso de build
    # propio), asi que el binario compilado se copia ahi tambien. Si cambias
    # wasmapp/main.go, hay que recompilar y volver a commitear
    # prototype/app.wasm — no hay CI todavia que lo automatice (ver
    # wasmapp/README.md, proximos pasos).
    $protoDir = Join-Path $PSScriptRoot "..\prototype"
    Copy-Item (Join-Path $outDir "app.wasm") (Join-Path $protoDir "app.wasm") -Force
    Copy-Item (Join-Path $outDir "wasm_exec.js") (Join-Path $protoDir "wasm_exec.js") -Force

    Write-Host "Listo:"
    Write-Host "  $outDir\app.wasm y wasm_exec.js (build de desarrollo)"
    Write-Host "  $protoDir\app.wasm y wasm_exec.js (copia para la app real)"
    Write-Host ""
    Write-Host "Para probar wasmapp/demo.html: servir wasmapp/ con un servidor estatico"
    Write-Host "(fetch() de .wasm no funciona con file://). Por ejemplo:"
    Write-Host "  python -m http.server 8000"
    Write-Host "  -> http://localhost:8000/demo.html"
}
finally {
    Pop-Location
}
