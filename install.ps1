$link = "https://github.com/Vencord/Installer/releases/latest/download/VencordInstallerCli.exe"

$outfile = "$env:TEMP\VencordInstallerCli.exe"

Write-Output "Downloading installer to $outfile"

Invoke-WebRequest -Uri "$link" -OutFile "$outfile"

Write-Output ""

Start-Process -Wait -NoNewWindow -FilePath "$outfile"

# Cleanup
Remove-Item -Force "$outfile"
