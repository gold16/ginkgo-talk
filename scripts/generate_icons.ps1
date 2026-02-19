param(
    [Parameter(Mandatory = $true)]
    [string]$Source,
    [string]$OutDir = "web"
)

Add-Type -AssemblyName System.Drawing

if (-not (Test-Path $Source)) {
    throw "Source image not found: $Source"
}

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
}

$srcImage = [System.Drawing.Image]::FromFile((Resolve-Path $Source))
try {
    $min = [Math]::Min($srcImage.Width, $srcImage.Height)
    $cropX = [int](($srcImage.Width - $min) / 2)
    $cropY = [int](($srcImage.Height - $min) / 2)

    foreach ($size in @(512, 192, 32)) {
        $bmp = New-Object System.Drawing.Bitmap($size, $size)
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.Clear([System.Drawing.Color]::Transparent)

        $srcRect = New-Object System.Drawing.Rectangle($cropX, $cropY, $min, $min)
        $dstRect = New-Object System.Drawing.Rectangle(0, 0, $size, $size)
        $g.DrawImage($srcImage, $dstRect, $srcRect, [System.Drawing.GraphicsUnit]::Pixel)

        $outPath = Join-Path $OutDir ("icon-{0}.png" -f $size)
        $bmp.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)

        $g.Dispose()
        $bmp.Dispose()
        Write-Host "Wrote $outPath"
    }
}
finally {
    $srcImage.Dispose()
}
