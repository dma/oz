package fs

import (
	"fmt"
	"os"
	"path"
	"syscall"
)

var basicBindDirs = []string{
	"/bin", "/lib", "/lib64", "/usr", "/etc", "/var/lib/oz",
}

var basicEmptyDirs = []string{
	"/sbin", "/var", "/var/lib",
	"/var/cache", "/home", "/boot",
	"/tmp", "/run", "/run/user",
	"/run/lock", "/root", "/opt",
	"/srv", "/dev", "/proc", "/sys",
	"/mnt", "/media",
}

var basicBlacklist = []string{
	"/usr/sbin", "/sbin", "${PATH}/su",
	"${PATH}/sudo", "${PATH}/fusermount",
	"${PATH}/xinput", "${PATH}/strace",
	"${PATH}/mount", "${PATH}/umount",
}

const emptyFilePath = "/tmp/oz.ro.file"
const emptyDirPath = "/tmp/oz.ro.dir"

var basicSymlinks = [][2]string{
	{"/run", "/var/run"},
	{"/tmp", "/var/tmp"},
	{"/run/lock", "/var/lock"},
}

func (fs *Filesystem) Setup() error {
	if err := fs.setupRootfs(); err != nil {
		return err
	}
	if err := fs.setupChroot(); err != nil {
		return err
	}
	return fs.setupMountItems()
}

func (fs *Filesystem) setupRootfs() error {
	if err := os.MkdirAll(fs.base, 0755); err != nil {
		return fmt.Errorf("unable to create directory (%s): %v", fs.base, err)
	}
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV)
	data := "mode=755,gid=0"
	if err := syscall.Mount(fs.base, fs.base, "tmpfs", flags, data); err != nil {
		return fmt.Errorf("failed to create base tmpfs at %s: %v", fs.base, err)
	}
	// create extra directories
	extra := []string{"sockets"}
	for _, sub := range extra {
		d := path.Join(fs.base, sub)
		if err := os.Mkdir(d, 0755); err != nil {
			return fmt.Errorf("unable to create directory (%s): %v", d, err)
		}
	}
	return nil
}

func (fs *Filesystem) setupChroot() error {
	var err error
	if fs.noDefaults {
		err = createEmptyDirectories(fs.root, basicBindDirs)
	} else {
		err = bindBasicDirectories(fs.root, basicBindDirs)
	}
	if err != nil {
		return err
	}
	err = createEmptyDirectories(fs.root, basicEmptyDirs)
	if err != nil {
		return err
	}
	return setupTmp(fs.root)
}

func bindBasicDirectories(root string, dirs []string) error {
	for _, src := range dirs {
		st, err := os.Lstat(src)
		if err != nil {
			return err
		}
		mode := st.Mode()
		target := path.Join(root, src)
		if err := os.MkdirAll(target, mode.Perm()); err != nil {
			return err
		}
		if err := bindMount(src, target, true, 0); err != nil {
			return err
		}
	}
	return nil
}

func createEmptyDirectories(root string, dirs []string) error {
	for _, p := range dirs {
		target := path.Join(root, p)
		if err := createEmptyDirectory(p, target); err != nil {
			return err
		}
	}
	return nil
}

func createEmptyDirectory(source, target string) error {
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if err := os.MkdirAll(target, mode.Perm()); err != nil {
		return err
	}
	if err := copyFileInfo(fi, target); err != nil {
		return err
	}
	return nil
}

func setupTmp(root string) error {
	target := path.Join(root, "tmp")
	if err := os.Chmod(target, 0777); err != nil {
		return err
	}
	return bindMount(target, target, false, syscall.MS_NOEXEC)
}

func (fs *Filesystem) setupMountItems() error {
	for _, item := range fs.whitelist {
		if err := item.bind(); err != nil {
			// XXX
		}
	}
	return nil
}
