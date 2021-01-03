#include <cino.h>
#include <Servo.h>

Servo myservo;

void setup()
{
    TEST_PLAN(2);
    CHECK(!myservo.attached());
    myservo.attach(9);
    CHECK(myservo.attached());
}

void loop() {}
