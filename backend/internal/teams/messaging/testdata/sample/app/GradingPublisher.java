package com.harvest.grading;

class GradingPublisher {
    void publish(Lot lot) {
        jms.convertAndSend("HL.LOT.GRADED", lot);
        kafka.send("grading.results", lot.id(), lot);
    }
}
