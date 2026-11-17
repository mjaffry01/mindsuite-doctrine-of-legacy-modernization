package com.harvest.settlement;

class SettlementListener {
    @JmsListener(destination = "HL.LOT.GRADED")
    void onGraded(Lot lot) { }

    void pay(Payment p) {
        jms.convertAndSend("HL.PAYMENT.OUT", p);
    }
}
