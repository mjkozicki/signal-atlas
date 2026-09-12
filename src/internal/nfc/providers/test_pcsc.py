"""Reader protocol tests use in-memory APDU responses; no hardware is accessed."""
import unittest
import pcsc

class FakeCard:
    def __init__(self, responses):
        self.responses = list(responses)
        self.commands = []
    def transmit(self, command):
        self.commands.append(bytes(command))
        if not self.responses:
            raise AssertionError('Unexpected APDU')
        return self.responses.pop(0)

class ReadProtocolTests(unittest.TestCase):
    def test_type4_select_and_read_only(self):
        message = bytes.fromhex('d101085402656e48656c6c6f')
        cc = bytes.fromhex('000f20003b00340406e104010000ff')
        card = FakeCard([( [],0x90,0),([],0x90,0),(list(cc),0x90,0),([],0x90,0),([0,len(message)],0x90,0),(list(message),0x90,0)])
        self.assertEqual(pcsc.type4_ndef(card), message)
        self.assertTrue(all(c[1] in (0xA4,0xB0) for c in card.commands))
    def test_read_respects_advertised_maximum(self):
        card = FakeCard([(list(bytes(15)),0x90,0),(list(bytes(5)),0x90,0)])
        self.assertEqual(pcsc.read_binary(card,2,20,max_read=15),bytes(20))
        self.assertEqual([c[-1] for c in card.commands],[15,5])

    def test_disconnect_after_atr_failure(self):
        from unittest.mock import Mock
        connection = Mock()
        connection.getATR.side_effect = RuntimeError('reader removed')
        reader = Mock()
        reader.createConnection.return_value = connection
        with self.assertRaises(RuntimeError): pcsc.inspect_reader(reader)
        connection.disconnect.assert_called_once()

    def test_bad_status_rejected(self):
        with self.assertRaises(ValueError):
            pcsc.apdu(FakeCard([([],0x69,0x82)]), [0,0xB0,0,0,1])
    def test_tlv_and_bounds(self):
        self.assertEqual(pcsc.extract_tlv(b'\x00\x03\x02hi\xfe'), b'hi')
        self.assertEqual(pcsc.extract_tlv(b'\x03\xff\x00\x02hi'), b'hi')
        for data in (b'\x03',b'\x03\x08hi',b'\x03\xff\x00'):
            with self.assertRaises(ValueError): pcsc.extract_tlv(data)
        with self.assertRaises(ValueError): pcsc.read_binary(FakeCard([]),0,4099)
    def test_type2_memory_read(self):
        cc=bytes.fromhex('e1100200')+bytes(12)
        memory=b'\x03\x02hi\xfe'+bytes(11)
        card=FakeCard([(list(cc),0x90,0),(list(memory),0x90,0)])
        self.assertEqual(pcsc.type2_ndef(card),b'hi')
        self.assertEqual(card.commands,[bytes.fromhex('ffb0000310'),bytes.fromhex('ffb0000410')])

if __name__ == '__main__': unittest.main()
