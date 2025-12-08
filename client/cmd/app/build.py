#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
自动打包脚本
"""

import os
import sys
import subprocess
import shutil
from create_icon import create_icon

def run_command(cmd, description):
    """执行命令并显示进度"""
    print(f"\n[{description}]")
    print(f"Command: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(cmd, check=True, capture_output=True, text=True)
        print(f"✓ {description} - Success")
        return True
    except subprocess.CalledProcessError as e:
        print(f"✗ {description} - Failed")
        print(f"Error: {e.stderr}")
        return False

def main():
    print("=" * 50)
    print("  StarFire MaaS - Auto Build Script")
    print("=" * 50)
    
    # Step 1: Install dependencies
    if not run_command(
        [sys.executable, "-m", "pip", "install", "pyinstaller", "pillow"],
        "Installing dependencies"
    ):
        print("\nFailed to install dependencies")
        return False
    
    # Step 1.5: Create icon
    print("\n[Creating icon]")
    try:
        create_icon()
        print("✓ Icon created successfully")
    except Exception as e:
        print(f"⚠ Icon creation failed: {e}")
        print("Continuing without custom icon...")
    
    # Step 2: Build executable
    build_cmd = [
        sys.executable, "-m", "PyInstaller",
        "--onefile",
        "--windowed",
        "--name", "StarFire_MaaS",
        "--clean",
        "--add-data", "starfire.exe;.",
        "--add-data", "icon.ico;.",
    ]
    
    # Add icon if it exists
    if os.path.exists("icon.ico"):
        build_cmd.extend(["--icon", "icon.ico"])
        print("✓ Using custom icon")
    else:
        print("⚠ No icon file found, using default")
    
    # Add the main script at the end
    build_cmd.append("star_fire.py")
    
    if not run_command(build_cmd, "Building executable"):
        print("\nBuild failed")
        return False
    
    # Step 3: Clean up
    print("\n[Cleaning up]")
    
    # Remove build directory
    if os.path.exists("build"):
        shutil.rmtree("build")
        print("✓ Removed build directory")
    
    # Remove __pycache__
    if os.path.exists("__pycache__"):
        shutil.rmtree("__pycache__")
        print("✓ Removed __pycache__")
    
    # Remove spec file
    if os.path.exists("StarFire_MaaS.spec"):
        os.remove("StarFire_MaaS.spec")
        print("✓ Removed spec file")
    
    # Copy starfire.exe to dist folder
    dist_starfire = os.path.join("dist", "starfire.exe")
    if os.path.exists("starfire.exe"):
        shutil.copy2("starfire.exe", dist_starfire)
        print("✓ Copied starfire.exe to dist folder")
    else:
        print("⚠ starfire.exe not found in source directory")
    
    # Step 4: Check output
    exe_path = os.path.join("dist", "StarFire_MaaS.exe")
    if os.path.exists(exe_path):
        size_mb = os.path.getsize(exe_path) / (1024 * 1024)
        print("\n" + "=" * 50)
        print("  BUILD COMPLETE!")
        print("=" * 50)
        print(f"\nOutput: {exe_path}")
        print(f"Size: {size_mb:.2f} MB")
        
        # Check if starfire.exe was copied
        starfire_dist = os.path.join("dist", "starfire.exe")
        if os.path.exists(starfire_dist):
            starfire_size = os.path.getsize(starfire_dist) / (1024 * 1024)
            print(f"StarFire Core: {starfire_size:.2f} MB")
        
        print("\nNext steps:")
        print("1. Navigate to dist folder")
        print("2. Run StarFire_MaaS.exe")
        print("3. StarFire core (starfire.exe) is included")
        return True
    else:
        print("\n✗ Build failed - executable not found")
        return False

if __name__ == "__main__":
    success = main()
    print("\n")
    input("Press Enter to exit...")
    sys.exit(0 if success else 1)