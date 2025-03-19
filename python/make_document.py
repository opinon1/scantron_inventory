#!/usr/bin/env python3
"""
Scantron-Style Document Generator for Inventory Management

This script generates an A4 PDF document that includes:
  - A Client section with a QR code (client id as a hex string)
  - A Date section with three fields:
      • Day: two-digit (first digit: 0–3, second digit: 0–9)
      • Month: two-digit (first digit: 0–1, second digit: 0–9)
      • Year: two-digit (decade and year, each 0–9)
    Each field is rendered as vertical columns of bubbles (highest digit at the top).
  - Multiple product rows; each row displays:
      • The product name (printed without extra spacing),
      • A QR code (from the product id),
      • An Inventory field for a two-digit number drawn with two horizontal bubble groups:
            – A “Tens” row (bubbles for 0–9 arranged horizontally)
            – An “Ones” row (bubbles for 0–9 arranged horizontally, to the right of “Tens”)
  - Orientation markers in three page corners (top left, top right, bottom left) and at the beginning of each product row.
  
Dependencies:
  - reportlab (install with: pip install reportlab)
"""

from reportlab.lib.pagesizes import A4
from reportlab.pdfgen import canvas
from reportlab.graphics.barcode import qr
from reportlab.graphics.shapes import Drawing
from reportlab.graphics import renderPDF

import random
import string

# -------------------------------
# Utility Drawing Functions
# -------------------------------
def id_generator(size=15, chars=string.ascii_uppercase + string.digits):
    return ''.join(random.choice(chars) for _ in range(size))

def draw_marker(c, x, y, size=10):
    """Draw a filled square marker at (x, y) with the given size."""
    c.rect(x, y, size, size, fill=1)

def draw_bubble_field_with_allowed(c, x, y, label, columns_allowed, bubble_radius=4, col_spacing=30, row_spacing=12):
    """
    Draw a vertical bubble field for a multi-digit number.
    
    Each column is drawn using the allowed digits (sorted descending so that the highest digit is on top).
    For example, for Day: first column allowed digits are [0,1,2,3] (drawn as [3,2,1,0]), and the second column [0–9].
    
    Parameters:
      c             : ReportLab canvas.
      x, y          : Top-left starting coordinates for the label.
      label         : Field label (e.g., "Day").
      columns_allowed: A list where each element is a list of allowed digits for that column.
      bubble_radius : Radius of each bubble.
      col_spacing   : Horizontal spacing between columns.
      row_spacing   : Vertical spacing between bubbles.
    """
    c.setFont("Helvetica-Bold", 8)
    c.drawString(x, y, label + ":")
    base_y = y - 15
    for col_index, allowed in enumerate(columns_allowed):
        allowed_sorted = sorted(allowed, reverse=True)
        for row_index, digit in enumerate(allowed_sorted):
            bubble_x = x + col_index * col_spacing
            bubble_y = base_y - row_index * row_spacing
            c.circle(bubble_x, bubble_y, bubble_radius, stroke=1, fill=0)
            c.drawString(bubble_x + bubble_radius + 2, bubble_y - bubble_radius/2, str(digit))

def draw_date_fields(c, x, y):
    """
    Draw the Day, Month, and Year fields.
    
    Day: two columns (first: allowed [0,1,2,3], second: allowed [0–9])
    Month: two columns (first: allowed [0,1], second: allowed [0–9])
    Year: two columns (each allowed [0–9])
    
    The fields are placed side by side with a horizontal gap.
    """
    gap = 100  # horizontal gap between fields
    
    # Day field
    day_columns = [ [0, 1, 2, 3], list(range(10)) ]
    draw_bubble_field_with_allowed(c, x, y, "Day", day_columns, bubble_radius=4, col_spacing=30, row_spacing=12)
    # Month field
    month_columns = [ [0, 1], list(range(10)) ]
    draw_bubble_field_with_allowed(c, x + gap, y, "Month", month_columns, bubble_radius=4, col_spacing=30, row_spacing=12)
    
    # Year field (two digits: decade and year)
    year_columns = [ list(range(10)), list(range(10)) ]
    draw_bubble_field_with_allowed(c, x + 2*gap, y, "Year", year_columns, bubble_radius=4, col_spacing=30, row_spacing=12)

def draw_horizontal_bubble_field(c, x, y, label, allowed_digits, bubble_radius=4, spacing=15):
    """
    Draw a horizontal row of bubbles.
    
    Parameters:
      c             : ReportLab canvas.
      x, y          : Starting coordinates for the label.
      label         : Field label (e.g., "Tens" or "Ones").
      allowed_digits: A list of allowed digits (typically 0–9).
      bubble_radius : Radius of each bubble.
      spacing       : Horizontal spacing between bubble centers.
    """
    c.setFont("Helvetica-Bold", 8)
    # c.drawString(x, y, label + ":")
    base_y = y - 15
    for i, digit in enumerate(allowed_digits):
        bubble_x = x + i * spacing
        c.circle(bubble_x, base_y, bubble_radius, stroke=1, fill=0)
        c.drawCentredString(bubble_x, base_y - bubble_radius - 8, str(digit))

# -------------------------------
# Main PDF Generation Function
# -------------------------------

def generate_scantron_pdf(client_id, client_name, products, output_filename):
    """
    Generate an A4 scantron document.
    
    Parameters:
      client_id      : A hex string representing the client id.
      products       : A list of dicts; each dict must have keys 'name' and 'id' for the product.
      output_filename: Name of the output PDF file.
    """
    c = canvas.Canvas(output_filename, pagesize=A4)
    page_width, page_height = A4
    marker_size = 10  # Marker size in points

    # Global orientation markers in three corners (omit bottom right)
    draw_marker(c, 0, 0, marker_size)                   # Bottom left
    draw_marker(c, 0, page_height - marker_size, marker_size)   # Top left
    draw_marker(c, page_width - marker_size, page_height - marker_size, marker_size)  # Top right

    # -------------------------------
    # Client Section
    # -------------------------------
    client_section_y = page_height - 50
    c.setFont("Helvetica-Bold", 14)
    c.drawString(50, client_section_y, "Client: " + client_name)

    # Draw client QR code using the client id.
    client_qr = qr.QrCodeWidget(client_id)
    bounds = client_qr.getBounds()
    width = bounds[2] - bounds[0]
    height = bounds[3] - bounds[1]
    qr_scale = 1.0  # Adjust this scale factor to change the QR code size if needed.
    d = Drawing(width * qr_scale, height * qr_scale)
    d.add(client_qr)
    d.scale(qr_scale, qr_scale)
    renderPDF.draw(d, c, 50, client_section_y - 85)

    # -------------------------------
    # Date Section (Day/Month/Year)
    # -------------------------------
    date_section_x = 300
    date_section_y = page_height - 50
    draw_date_fields(c, date_section_x, date_section_y)

    # -------------------------------
    # Product Rows Section
    # -------------------------------
    # Begin product rows lower on the page and use reduced padding
    product_start_y = page_height - 200
    row_spacing = 30  # Thinner rows

    qr_scale = 0.3  # Adjust this scale factor to change the QR code size if needed.
    for idx, product in enumerate(products):
        current_y = product_start_y - idx * row_spacing

        # Orientation marker for each product row
        draw_marker(c, 10, current_y - 10, marker_size)

        # Print product name without extra spaces (aligned closer to the left)
        c.setFont("Helvetica", 12)
        c.drawString(30, current_y -10, product['name'])

        # Draw product QR code using the product id.
        prod_qr = qr.QrCodeWidget(product['id'])
        bounds = prod_qr.getBounds()
        width = bounds[2] - bounds[0]
        height = bounds[3] - bounds[1]
        d = Drawing(width * qr_scale, height * qr_scale)
        d.add(prod_qr)
        d.scale(qr_scale, qr_scale)
        renderPDF.draw(d, c, 160, current_y - 20)

        # Draw Inventory fields for the product.
        # Inventory bubbles drawn as horizontal rows with reduced spacing,
        # moved to the left.
        inv_field_x = 200  # Starting x for inventory fields (moved left)
        draw_horizontal_bubble_field(c, inv_field_x, current_y + 13, "Tens", list(range(10)), bubble_radius=4, spacing=15)
        tens_width = 10 * 15  # 10 bubbles at 15 spacing each
        gap_between = 10
        draw_horizontal_bubble_field(c, inv_field_x + tens_width + gap_between, current_y + 13, "Ones", list(range(10)), bubble_radius=4, spacing=15)

    # Save the PDF
    c.save()
    print(f"PDF generated and saved as '{output_filename}'.")

# -------------------------------
# Example Usage
# -------------------------------
if __name__ == "__main__":
    # Example client id (hex string)
    client_id = id_generator()
    client_name = "Rodoltte"
    # Example products; add as many as needed
    products = [

{"name": "Croissant de almendra", "id": "Croissant de almendra"},
{"name": "Croissant tradicional", "id": "Croissant tradicional"},
{"name": "Croissant de chocolate", "id": "Croissant de chocolate"},
{"name": "Rol de Canela", "id": "Rol de Canela"},
{"name": "Rol de guayaba", "id": "Rol de guayaba"},
{"name": "Rol de higo", "id": "Rol de higo"},
{"name": "Cartera de Manzana", "id": "Cartera de Manzana"},
{"name": "Cartera de durazno", "id": "Cartera de durazno"},
{"name": "Cart crema pastel", "id": "Cartera de crema pastelera"},
{"name": "Concha de cafe", "id": "Concha de cafe"},
{"name": "Concha de chocolate", "id": "Concha de chocolate"},
{"name": "Concha de vainilla", "id": "Concha de vainilla"},
{"name": "Cruffin de mango", "id": "Cruffin de mango"},
{"name": "Cruffin de frambuesa", "id": "Cruffin de frambuesa"},
{"name": "Nudo de cardamomo", "id": "Nudo de cardamomo"},
{"name": "Cono crema muselina", "id": "Cono de crema muselina"},
{"name": "Trenza crema pastelera", "id": "Trenza de crema pastelera"},
{"name": "Ocho crema pastelera", "id": "Ocho de crema pastelera"}


    ]
    output_filename = "inventory_scantron.pdf"
    generate_scantron_pdf(client_id, client_name, products, output_filename)
