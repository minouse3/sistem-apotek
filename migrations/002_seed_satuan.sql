-- Default packaging and base units for the medicine catalog.

INSERT INTO satuan (nama_satuan)
VALUES
    ('tablet'),
    ('kaplet'),
    ('kapsul'),
    ('strip'),
    ('blister'),
    ('box'),
    ('botol'),
    ('tube'),
    ('sachet'),
    ('ampul'),
    ('vial'),
    ('pcs')
ON CONFLICT (nama_satuan) DO NOTHING;
