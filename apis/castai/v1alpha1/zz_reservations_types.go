/*
Copyright 2022 Upbound Inc.
*/

// Code generated by upjet. DO NOT EDIT.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

type ReservationsObservation struct {
	ID *string `json:"id,omitempty" tf:"id,omitempty"`

	// organization
	OrganizationID *string `json:"organizationId,omitempty" tf:"organization_id,omitempty"`

	Reservations []ReservationsReservationsObservation `json:"reservations,omitempty" tf:"reservations,omitempty"`

	// csv file containing reservations
	ReservationsCsv *string `json:"reservationsCsv,omitempty" tf:"reservations_csv,omitempty"`
}

type ReservationsParameters struct {

	// organization
	// +kubebuilder:validation:Optional
	OrganizationID *string `json:"organizationId,omitempty" tf:"organization_id,omitempty"`

	// csv file containing reservations
	// +kubebuilder:validation:Optional
	ReservationsCsv *string `json:"reservationsCsv,omitempty" tf:"reservations_csv,omitempty"`
}

type ReservationsReservationsObservation struct {
	Count *string `json:"count,omitempty" tf:"count,omitempty"`

	EndDate *string `json:"endDate,omitempty" tf:"end_date,omitempty"`

	InstanceType *string `json:"instanceType,omitempty" tf:"instance_type,omitempty"`

	Name *string `json:"name,omitempty" tf:"name,omitempty"`

	Price *string `json:"price,omitempty" tf:"price,omitempty"`

	Provider *string `json:"provider,omitempty" tf:"provider,omitempty"`

	Region *string `json:"region,omitempty" tf:"region,omitempty"`

	StartDate *string `json:"startDate,omitempty" tf:"start_date,omitempty"`

	ZoneID *string `json:"zoneId,omitempty" tf:"zone_id,omitempty"`

	ZoneName *string `json:"zoneName,omitempty" tf:"zone_name,omitempty"`
}

type ReservationsReservationsParameters struct {
}

// ReservationsSpec defines the desired state of Reservations
type ReservationsSpec struct {
	v1.ResourceSpec `json:",inline"`
	ForProvider     ReservationsParameters `json:"forProvider"`
}

// ReservationsStatus defines the observed state of Reservations.
type ReservationsStatus struct {
	v1.ResourceStatus `json:",inline"`
	AtProvider        ReservationsObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Reservations is the Schema for the Reservationss API. <no value>
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,castai}
type Reservations struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self.managementPolicy == 'ObserveOnly' || has(self.forProvider.reservationsCsv)",message="reservationsCsv is a required parameter"
	Spec   ReservationsSpec   `json:"spec"`
	Status ReservationsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReservationsList contains a list of Reservationss
type ReservationsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Reservations `json:"items"`
}

// Repository type metadata.
var (
	Reservations_Kind             = "Reservations"
	Reservations_GroupKind        = schema.GroupKind{Group: CRDGroup, Kind: Reservations_Kind}.String()
	Reservations_KindAPIVersion   = Reservations_Kind + "." + CRDGroupVersion.String()
	Reservations_GroupVersionKind = CRDGroupVersion.WithKind(Reservations_Kind)
)

func init() {
	SchemeBuilder.Register(&Reservations{}, &ReservationsList{})
}