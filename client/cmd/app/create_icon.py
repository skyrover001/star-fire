#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
创建一个简单的应用图标
"""

from PIL import Image, ImageDraw, ImageFont
import os

def create_icon():
    """创建一个简单的图标"""
    # 创建 256x256 的图像，使用RGB模式（Windows图标标准）
    size = 256
    
    # 先创建RGBA图像以支持抗锯齿
    img_rgba = Image.new('RGBA', (size, size), color=(255, 255, 255, 0))
    draw = ImageDraw.Draw(img_rgba)
    
    # 绘制一个星星和火焰的简单图案
    # 绘制圆形背景 - 更鲜艳的橙红色
    draw.ellipse([20, 20, 236, 236], fill='#FF4500', outline='#FF6347', width=8)
    
    # 绘制星星（简化版）- 更亮的黄色
    star_points = [
        (128, 60), (150, 115), (210, 115), (165, 150),
        (185, 200), (128, 165), (71, 200), (91, 150),
        (46, 115), (106, 115)
    ]
    draw.polygon(star_points, fill='#FFD700', outline='#FFA500', width=3)
    
    # 在中心添加高光效果
    draw.ellipse([115, 105, 141, 131], fill='#FFFF00')
    
    # 保存为 PNG
    img_rgba.save('icon.png', 'PNG')
    print("✓ 图标已创建: icon.png")
    
    # 转换为ICO - 为Windows创建标准尺寸
    try:
        # 创建白色背景的RGB图像（Windows标准）
        img_rgb = Image.new('RGB', (size, size), color=(255, 255, 255))
        img_rgb.paste(img_rgba, (0, 0), img_rgba)
        
        # 生成多尺寸图标
        icon_sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)]
        
        # 保存ICO文件
        img_rgb.save('icon.ico', format='ICO', sizes=icon_sizes)
        print("✓ ICO 图标已创建: icon.ico")
        
        # 验证文件大小
        import os
        size_kb = os.path.getsize('icon.ico') / 1024
        print(f"  图标文件大小: {size_kb:.1f} KB")
        
    except Exception as e:
        print(f"⚠ 创建 ICO 失败: {e}")
        print("提示: 运行 'pip install pillow' 安装依赖")

if __name__ == "__main__":
    try:
        create_icon()
    except ImportError:
        print("请先安装 Pillow: pip install pillow")