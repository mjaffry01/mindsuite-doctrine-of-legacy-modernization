"""Deterministic documents sample: Word, PowerPoint, Visio, PDF (text and scanned), Confluence HTML,
Markdown and draw.io (plain and compressed), with known rules, terms, processes and links."""
import os, zipfile, zlib, base64, urllib.parse
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'sample') + '/'
os.makedirs(D, exist_ok=True)
W = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"'


def wp(text, style=None):
    st = f'<w:pPr><w:pStyle w:val="{style}"/></w:pPr>' if style else ''
    return f'<w:p>{st}<w:r><w:t xml:space="preserve">{text}</w:t></w:r></w:p>'


def wrow(*cells):
    return '<w:tr>' + ''.join(f'<w:tc>{wp(c)}</w:tc>' for c in cells) + '</w:tr>'


body = ''.join([
    wp('Settlement specification', 'Heading1'),
    wp('Last updated: 12 March 2018'),
    wp('Version 2.1'),
    wp('Business rules', 'Heading2'),
    wp('A grower must be paid within 14 days of delivery.'),
    wp('Deliveries with moisture above 18% are not allowed for grade A.'),
    wp('Payments over 50,000 EUR require approval by the finance manager, and are always sent in the next payment run.'),
    wp('Glossary', 'Heading2'),
    '<w:tbl>' + wrow('Term', 'Meaning') + wrow('Lot', 'A batch of fruit delivered by one grower on one day') + wrow('Grower', 'A farmer who delivers fruit to the intake') + '</w:tbl>',
    wp('Settlement process', 'Heading2'),
    wp('1. Receive the delivery at the intake'),
    wp('2. Grade the lot'),
    wp('3. Calculate the payment'),
    wp('4. Send the payment file to the bank'),
    wp('TBD: how returned fruit is credited.'),
])
with zipfile.ZipFile(D + 'settlement-spec.docx', 'w') as z:
    z.writestr('[Content_Types].xml', '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>')
    z.writestr('word/document.xml', f'<?xml version="1.0" encoding="UTF-8"?><w:document {W}><w:body>{body}</w:body></w:document>')

A = 'xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"'
def slide(*paras):
    return f'<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" {A}><p:cSld><p:spTree>' + ''.join(f'<p:sp><p:txBody><a:p><a:r><a:t>{t}</a:t></a:r></a:p></p:txBody></p:sp>' for t in paras) + '</p:spTree></p:cSld></p:sld>'
with zipfile.ZipFile(D + 'overview.pptx', 'w') as z:
    z.writestr('ppt/slides/slide1.xml', slide('HarvestLink overview', 'Growers deliver fruit to the intake'))
    z.writestr('ppt/slides/slide2.xml', slide('Payments', 'Payments run every night and the bank receives one file'))

with zipfile.ZipFile(D + 'system-context.vsdx', 'w') as z:
    z.writestr('visio/pages/page1.xml', '''<?xml version="1.0"?><PageContents xmlns="http://schemas.microsoft.com/office/visio/2012/main"><Shapes>
<Shape ID="1"><Text>ERP</Text></Shape><Shape ID="2"><Text>HarvestLink</Text></Shape><Shape ID="3"><Text>invoices</Text></Shape>
</Shapes><Connects><Connect FromSheet="3" FromCell="BeginX" ToSheet="2"/><Connect FromSheet="3" FromCell="EndX" ToSheet="1"/></Connects></PageContents>''')


def pdf(stream: bytes, compress: bool) -> bytes:
    data = zlib.compress(stream) if compress else stream
    filt = b' /Filter /FlateDecode' if compress else b''
    return (b'%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n'
            b'3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj\n4 0 obj << /Length ' + str(len(data)).encode() + filt + b' >>\nstream\n' + data + b'\nendstream\nendobj\ntrailer << /Root 1 0 R >>\n%%EOF\n')


text = b'BT /F1 12 Tf 72 720 Td (Payment policy) Tj 0 -16 Td (Date: 2025-06-30) Tj 0 -16 Td (Every payment must be approved before it is sent to the bank.) Tj 0 -16 Td (Refunds are only allowed within 30 days of the payment.) Tj ET'
open(D + 'payment-policy.pdf', 'wb').write(pdf(text, True))
open(D + 'scanned-contract.pdf', 'wb').write(pdf(b'q 612 0 0 792 0 0 cm /Im1 Do Q', False))
open(D + 'old-manual.doc', 'wb').write(b'\xd0\xcf\x11\xe0legacy word file')

open(D + 'operations-handbook.html', 'w', newline='').write('''<html><head><title>Operations handbook</title></head><body>
<h1>Operations handbook</h1>
<p>Last updated: 2025-09-01</p>
<h2>Glossary</h2>
<table>
<tr><td>Lot</td><td>One truck load of fruit, whoever grew it</td></tr>
</table>
<h2>Month-end process</h2>
<ol>
<li>Close the period in the settlement system</li>
<li>Run the grower statements</li>
<li>Send the ledger file to finance</li>
</ol>
<h2>Architecture</h2>
<p>The portal calls the settlement service over REST; settlement stores its data in Oracle and sends payment files over SFTP.</p>
</body></html>
''')

open(D + 'data-dictionary.md', 'w', newline='').write('''# Data dictionary

Moisture means the water content of a lot, measured at the intake.
Each delivery gets a Goods Received Note (GRN) at the scale.
''')

open(D + 'architecture.drawio', 'w', newline='').write('''<mxfile host="drawio">
  <diagram name="Architecture" id="a1">
    <mxGraphModel><root>
      <mxCell id="0"/><mxCell id="1" parent="0"/>
      <mxCell id="p" value="Grower portal" vertex="1" parent="1"/>
      <mxCell id="s" value="Settlement &lt;br&gt;service" vertex="1" parent="1"/>
      <mxCell id="o" value="Oracle DB" vertex="1" parent="1"/>
      <mxCell id="b" value="Partner Bank" vertex="1" parent="1"/>
      <mxCell id="e1" value="REST" edge="1" source="p" target="s" parent="1"/>
      <mxCell id="e2" edge="1" source="s" target="o" parent="1"/>
      <mxCell id="e3" value="SFTP" edge="1" source="s" target="b" parent="1"/>
    </root></mxGraphModel>
  </diagram>
</mxfile>
''')

inner = '<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="d" value="Delivery" vertex="1" parent="1"/><mxCell id="g" value="Grading" vertex="1" parent="1"/><mxCell id="x" edge="1" source="d" target="g" parent="1"/></root></mxGraphModel>'
co = zlib.compressobj(9, zlib.DEFLATED, -15)
packed = base64.b64encode(co.compress(urllib.parse.quote(inner).encode()) + co.flush()).decode()
open(D + 'intake-flow.drawio', 'w', newline='').write(f'<mxfile host="drawio">\n  <diagram name="Intake" id="f1">{packed}</diagram>\n</mxfile>\n')

open(D + 'notes.properties', 'w', newline='').write('sharepoint.password=SharePointPw2026\n')
print('sample ok')
