param(
    [Parameter(Mandatory = $true)]
    [string]$Source,
    [string]$OutPath = ""
)

Add-Type -AssemblyName System.Drawing

if (-not (Test-Path $Source)) {
    throw "Source image not found: $Source"
}

$srcPath = (Resolve-Path $Source).Path
$src = [System.Drawing.Bitmap]::new($srcPath)

try {
    # Create a new 32bpp ARGB bitmap to support transparency
    $w = $src.Width
    $h = $src.Height
    $dst = New-Object System.Drawing.Bitmap($w, $h, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)

    for ($y = 0; $y -lt $h; $y++) {
        for ($x = 0; $x -lt $w; $x++) {
            $c = $src.GetPixel($x, $y)
            # Calculate how "white" the pixel is
            $minChannel = [Math]::Min([Math]::Min([int]$c.R, [int]$c.G), [int]$c.B)
            $brightness = ($c.R + $c.G + $c.B) / 3.0

            if ($brightness -gt 248 -and $minChannel -gt 240) {
                # Very close to pure white -> fully transparent
                $dst.SetPixel($x, $y, [System.Drawing.Color]::FromArgb(0, 255, 255, 255))
            }
            elseif ($brightness -gt 230 -and $minChannel -gt 220) {
                # Near-white -> partially transparent (smooth edge)
                $factor = ($brightness - 230) / 25.0
                $alpha = [int]([Math]::Max(0, [Math]::Min(255, 255 * (1.0 - $factor))))
                $dst.SetPixel($x, $y, [System.Drawing.Color]::FromArgb($alpha, $c.R, $c.G, $c.B))
            }
            else {
                # Keep the pixel as-is
                $dst.SetPixel($x, $y, $c)
            }
        }
        if ($y % 50 -eq 0) {
            Write-Host "Processing row $y / $h ..."
        }
    }

    if ($OutPath -eq "") {
        $dir = [System.IO.Path]::GetDirectoryName($srcPath)
        $name = [System.IO.Path]::GetFileNameWithoutExtension($srcPath)
        $OutPath = Join-Path $dir ($name + "_transparent.png")
    }

    $dst.Save($OutPath, [System.Drawing.Imaging.ImageFormat]::Png)
    Write-Host "Saved transparent image to: $OutPath"
}
finally {
    $src.Dispose()
    if ($dst) { $dst.Dispose() }
}
