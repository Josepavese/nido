#!/usr/bin/env python3
"""Package retention must preserve every architecture of each retained release."""
import importlib.util
import pathlib
import tempfile
import unittest

SOURCE = pathlib.Path(__file__).resolve().parents[1]/'bin/prune-package-versions.py'
spec = importlib.util.spec_from_file_location('package_retention', SOURCE)
retention = importlib.util.module_from_spec(spec)
spec.loader.exec_module(retention)


class PackageRetention(unittest.TestCase):
    def test_keeps_latest_versions_with_both_architectures(self):
        for extension, architectures in [('deb', ['amd64', 'arm64']),
                                          ('rpm', ['x86_64', 'aarch64'])]:
            with self.subTest(extension=extension), tempfile.TemporaryDirectory() as folder:
                root = pathlib.Path(folder)
                names = {}
                for version in ['4.8.7', '4.9.0', '4.10.0', '4.10.1', '4.10.2', '4.11.0']:
                    names[version] = []
                    for arch in architectures:
                        name = (f'nido_{version}_{arch}.deb' if extension == 'deb'
                                else f'nido-{version}-1.{arch}.rpm')
                        (root/name).write_bytes(b'package fixture')
                        names[version].append(name)
                retention.prune(root, '*.'+extension, 5)
                expected = {name for version in names if version != '4.8.7'
                            for name in names[version]}
                self.assertEqual({p.name for p in root.iterdir()}, expected)

    def test_unknown_filename_refuses_pruning_without_deleting_packages(self):
        with tempfile.TemporaryDirectory() as folder:
            root = pathlib.Path(folder)
            for name in ['unknown.rpm', 'nido-4.10.1-1.x86_64.rpm',
                         'nido-4.10.2-1.aarch64.rpm']:
                (root/name).touch()
            before = set(root.iterdir())
            with self.assertRaises(ValueError):
                retention.prune(root, '*.rpm', 1)
            self.assertEqual(set(root.iterdir()), before)


if __name__ == '__main__':
    unittest.main()
