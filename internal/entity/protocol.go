package entity

import (
	"gorm.io/gorm"
)

// configured sizes are for mysql, since version 5 mysql counts characters, not bytes

type ProtocolEntry struct {
	gorm.Model
	ReferenceId string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;index:nexi_ref_id_idx"`
	ApiId       string
	Kind        string `gorm:"type:varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	Message     string `gorm:"type:varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Details     string `gorm:"type:longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`   // usually: json message
	RequestId   string `gorm:"type:varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"` // optional
}
