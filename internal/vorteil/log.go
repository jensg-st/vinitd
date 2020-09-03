package vorteil

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	vlog logFn
)

const (
	MSG_IOCTL_OUTPUT       = 0x40042101
	MSG_IOCTL_OUTPUT_STYLE = 0x40042102
)

func logAlways(format string, values ...interface{}) {
	// write to stderr and kernel logs
	vlog(LOG_STDERR, format, values...)
	vlog(LOG_DEBUG, format, values...)
}

func logDebug(format string, values ...interface{}) {
	vlog(LOG_DEBUG, format, values...)
}

func logWarn(format string, values ...interface{}) {
	vlog(LOG_WARNING, format, values...)
}

func SystemPanic(format string, values ...interface{}) {
	logAlways(format, values...)
	shutdown(syscall.LINUX_REBOOT_CMD_POWER_OFF, forcedPoweroffTimeout)
}

func logError(format string, values ...interface{}) {
	vlog(LOG_STDERR, format, values...)
}

func writeToOut(out *os.File, format string, values ...interface{}) {
	txt := fmt.Sprintf(format, values...)
	up := fmt.Sprintf("[%05.6f]", uptime())
	fmt.Fprintf(out, "%s %s\n", up, txt)
	out.Sync()
}

// LogFnStdout prints all messages to stdout for testing
func LogFnStdout(level LogLevel, format string, values ...interface{}) {
	writeToOut(os.Stdout, format, values...)
}

/* LogFnKernel prints messages to /dev/kmsg. Based on the kernel's LogLevel
   messages will appear on stdout. LOG_STDERR always prints to screen independent
   of log level */
func LogFnKernel(level LogLevel, format string, values ...interface{}) {
	if level == LOG_STDERR {
		writeToOut(os.Stderr, format, values...)
	} else {

		txt := fmt.Sprintf("<%d>%s", level, fmt.Sprintf(format, values...))
		f, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()

		_, err = f.Write([]byte(txt))
		if err != nil {
			return
		}

	}
}

func printVersion() error {

	pv, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return err
	}

	s := strings.Split(string(pv), "(")
	version := s[0]

	kv, err := ioutil.ReadFile("/proc/sys/kernel/version")
	if err != nil {
		return err
	}

	logAlways("%s (%s)", strings.TrimSpace(string(kv)), strings.TrimSpace(version))

	return nil
}

func setupVtty(mode int) {

	file, err := os.OpenFile(defaultTTY, os.O_RDWR, 0)
	if err != nil {
		LogFnKernel(LOG_ERR, "can not open vtty: %s", err.Error())
	}
	defer file.Close()

	_, _, ep := unix.Syscall(unix.SYS_IOCTL, file.Fd(),
		MSG_IOCTL_OUTPUT, uintptr(unsafe.Pointer(&mode)))
	if ep != 0 {
		if err != nil {
			LogFnKernel(LOG_ERR, "can not ioctl vtty: %s", err.Error())
		}
	}

	os.Stdout, err = os.OpenFile(defaultTTY,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logWarn("can not assign /dev/vtty to vinitd")
	}

	os.Stderr, err = os.OpenFile(defaultTTY,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logWarn("can not assign /dev/vtty to vinitd")
	}

}