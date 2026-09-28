//go:build windows

package agent

import (
	"encoding/json"
	"os/exec"
)

func applicationUsage24Hours() (map[string]int64, error) {
	script := `$events=Get-WinEvent -FilterHashtable @{LogName='Security';Id=4688,4689;StartTime=(Get-Date).AddHours(-24)} -ErrorAction Stop | Sort-Object TimeCreated;$running=@{};$totals=@{};foreach($e in $events){$x=[xml]$e.ToXml();$d=@{};foreach($i in $x.Event.EventData.Data){$d[$i.Name]=$i.'#text'};if($e.Id -eq 4688){$pid=$d['NewProcessId'];$name=[IO.Path]::GetFileNameWithoutExtension($d['NewProcessName']);if($pid -and $name){$running[$pid]=@($name,$e.TimeCreated)}}elseif($e.Id -eq 4689){$pid=$d['ProcessId'];if($running.ContainsKey($pid)){$r=$running[$pid];$s=[int64]($e.TimeCreated-$r[1]).TotalSeconds;if($s -gt 0){$totals[$r[0]]=[int64]$totals[$r[0]]+$s};$running.Remove($pid)}}};$now=Get-Date;foreach($r in $running.Values){$s=[int64]($now-$r[1]).TotalSeconds;if($s -gt 0){$totals[$r[0]]=[int64]$totals[$r[0]]+$s}};$totals | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	usage := map[string]int64{}
	if len(out) == 0 || string(out) == "null\r\n" {
		return usage, nil
	}
	return usage, json.Unmarshal(out, &usage)
}
