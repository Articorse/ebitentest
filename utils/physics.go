package utils

func CalculateImpulse(velocity Vec2, normal Vec2, restitution float64) Vec2 {
	velocityAlongNormal := velocity.Dot(normal)
	if velocityAlongNormal >= 0 {
		return Vec2{}
	}
	impulseMagnitude := -(1 + restitution) * velocityAlongNormal
	return normal.Multiply(impulseMagnitude)
}
