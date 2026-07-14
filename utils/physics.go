package utils

func CalculateImpulse(velocity Vec2f, normal Vec2f, restitution float64) Vec2f {
	velocityAlongNormal := velocity.Dot(normal)
	if velocityAlongNormal >= 0 {
		return Vec2f{}
	}
	impulseMagnitude := -(1 + restitution) * velocityAlongNormal
	return normal.Multiply(impulseMagnitude)
}
