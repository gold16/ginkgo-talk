# Convert a PNG file to ICO format (PNG-compressed ICO, supported on Windows Vista+)
param(
    [Parameter(Mandatory = $true)]
    [string]$Source,
    [string]$OutPath = ""
)

if (-not (Test-Path $Source)) {
    throw "Source image not found: $Source"
}

$srcPath = (Resolve-Path $Source).Path
$pngBytes = [System.IO.File]::ReadAllBytes($srcPath)

Add-Type -AssemblyName System.Drawing
$img = [System.Drawing.Image]::FromFile($srcPath)
$width = $img.Width
$height = $img.Height
$img.Dispose()

# ICO with PNG payload
if ($OutPath -eq "") {
    $dir = [System.IO.Path]::GetDirectoryName($srcPath)
    $name = [System.IO.Path]::GetFileNameWithoutExtension($srcPath)
    $OutPath = Join-Path $dir ($name + ".ico")
}

$ms = New-Object System.IO.MemoryStream

# ICO Header (6 bytes)
$ms.Write([byte[]]@(0, 0), 0, 2)        # Reserved
$ms.Write([System.BitConverter]::GetBytes([uint16]1), 0, 2)  # Type = 1 (ICO)
$ms.Write([System.BitConverter]::GetBytes([uint16]1), 0, 2)  # Count = 1

# ICO Directory Entry (16 bytes)
# Width & Height: use 0 if 256, otherwise actual value; for sizes > 255, use 0
$w = if ($width -ge 256) { 0 } else { $width }
$h = if ($height -ge 256) { 0 } else { $height }
$ms.WriteByte([byte]$w)          # Width
$ms.WriteByte([byte]$h)          # Height
$ms.WriteByte(0)                 # Color palette count (0 = no palette)
$ms.WriteByte(0)                 # Reserved
$ms.Write([System.BitConverter]::GetBytes([uint16]1), 0, 2)   # Color planes
$ms.Write([System.BitConverter]::GetBytes([uint16]32), 0, 2)  # Bits per pixel
$ms.Write([System.BitConverter]::GetBytes([uint32]$pngBytes.Length), 0, 4) # Image size
$ms.Write([System.BitConverter]::GetBytes([uint32]22), 0, 4)  # Offset (6 + 16 = 22)

# PNG payload
$ms.Write($pngBytes, 0, $pngBytes.Length)

[System.IO.File]::WriteAllBytes($OutPath, $ms.ToArray())
$ms.Dispose()

Write-Host "Created ICO: $OutPath ($width x $height, $($pngBytes.Length) bytes PNG payload)"
