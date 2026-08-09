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

    Write-Host "Listo:"
    Write-Host "  $outDir\app.wasm"
    Write-Host "  $outDir\wasm_exec.js"
    Write-Host ""
    Write-Host "Para probar: servir wasmapp/ con un servidor estatico (fetch() de .wasm"
    Write-Host "no funciona con file://) y abrir demo.html. Por ejemplo:"
    Write-Host "  python -m http.server 8000"
    Write-Host "  -> http://localhost:8000/demo.html"
}
finally {
    Pop-Location
}
