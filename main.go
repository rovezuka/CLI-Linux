package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/process"
	"github.com/urfave/cli"
)

// Структура представляет информацию о дисковых разделах
type Volume struct {
	Name       string
	Total      uint64
	Used       uint64
	Available  uint64
	UsePercent float64
	Mount      string
}

// Функция-обработчик для выполнения действий с объемами
func ActionVolumes(c *cli.Context) error {

	// Возвращает статистику о дисковых разделах
	stats, err := disk.Partitions(true)
	if err != nil {
		return err
	}

	var vols []*Volume

	for _, stat := range stats {
		// Информация о использовании дискового пространства
		usage, err := disk.Usage(stat.Mountpoint)
		if err != nil {
			continue
		}

		vol := &Volume{
			Name:       stat.Device,
			Total:      usage.Total,
			Used:       usage.Used,
			Available:  usage.Free,
			UsePercent: usage.UsedPercent,
		}

		vols = append(vols, vol)
	}

	volsByteArr, err := json.MarshalIndent(vols, "", "\t")
	if err != nil {
		return err
	}

	fmt.Println(string(volsByteArr))
	return nil
}

// Если нет аргумента, идентификатора или имени, она вернет ошибку
func KillAction(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return errors.New("no arguments is expected, use flags")
	}

	if c.IsSet("id") && c.IsSet("name") {
		return errors.New("either pid or name flag must be provided")
	}

	if !c.IsSet("id") && c.String("name") == "" {
		return errors.New("name flag cannot be empty")
	}

	if err := killProcess(c); err != nil {
		return err
	}
	fmt.Println("Process killed successfully.")
	return nil
}

/*
Если идентификатор задан, мы возьмем процессы и уничтожим их, если идентификаторы совпадают.

Затем мы проделываем то же самое с именем.

Но здесь мы сравниваем со строками.

В два раза больше, чтобы устранить проблему с заглавными буквами в Windows.
*/
func killProcess(c *cli.Context) error {
	if c.IsSet("id") {
		proc, err := process.NewProcess(int32(c.Uint("id")))
		if err != nil {
			return err
		}

		return proc.Kill()
	}

	processes, err := process.Processes()
	if err != nil {
		return err
	}

	var (
		errs  []string
		found bool
	)

	target := c.String("name")
	for _, p := range processes {
		name, _ := p.Name()
		if name == "" {
			continue
		}

		if isEqualProcessName(name, target) {
			found = true
			if err := p.Kill(); err != nil {
				e := err.Error()
				errs = append(errs, e)
			}
		}
	}

	if !found {
		return errors.New("process not found")
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}

func isEqualProcessName(proc1 string, proc2 string) bool {
	if runtime.GOOS == "linux" {
		return proc1 == proc2
	}
	return strings.EqualFold(proc1, proc2)
}

func main() {
	// Создаем новое CLI приложение, используя NewApp, и запускаем приложение, вызывая Run
	app := cli.NewApp()
	app.Name = "Basic Kill and Delete Command İmplementation CLI"
	app.Usage = "Let's you kill processes by name or id and delete files or folders"

	app.Commands = []cli.Command{
		{
			Name:        "kill",
			HelpName:    "kill",
			Action:      KillAction,
			ArgsUsage:   ` `,
			Usage:       `kills processes by process id or process name.`,
			Description: `Terminate a process.`,
			Flags: []cli.Flag{
				&cli.UintFlag{
					Name:  "id",
					Usage: "kill process by process ID.",
				},
				&cli.StringFlag{
					Name:  "name",
					Usage: "kill process by process name. ",
				},
			},
		},
		{
			Name:        "volumes",
			HelpName:    "volumes",
			Action:      ActionVolumes,
			ArgsUsage:   `  `,
			Usage:       `lists mounted file system volumes.`,
			Description: `List the mounted volumes.`,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
