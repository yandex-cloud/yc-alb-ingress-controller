package k8s

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	maxAbsOrder    = 100000
	ControllerName = "ingress.alb.yc.io/yc-alb-ingress-controller"
)

type IngressGroup struct {
	Tag     string
	Items   []v1.Ingress
	Deleted []v1.Ingress
}

type Loader struct {
	cli client.Client
}

func NewGroupLoader(cli client.Client) *Loader {
	return &Loader{cli: cli}
}

func (l *Loader) Load(ctx context.Context, nsName types.NamespacedName) (*IngressGroup, error) {
	var list v1.IngressList
	err := l.cli.List(ctx, &list)
	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, nil
	}
	tag := nsName.Name

	if len(tag) == 0 {
		return nil, nil
	}

	retItems := make([]ingressWithOrder, 0)
	deletedItems := make([]v1.Ingress, 0)

	var classList v1.IngressClassList
	err = l.cli.List(ctx, &classList)
	if err != nil {
		return nil, err
	}

	for _, item := range list.Items {
		managed := IsIngressManagedByThisController(item, classList)
		hasfinalizer := hasFinalizer(&item, Finalizer)
		deleted := !item.GetDeletionTimestamp().IsZero()

		// belongs to other group -> skip
		if managed && !belongsToGroup(item, tag) {
			continue
		}

		// belongs to the group -> use
		if managed && !deleted {
			prior, err := parseOrder(item.GetAnnotations()[OrderInGroup])
			if err != nil {
				return nil, fmt.Errorf("error getting order of ingress %s: %e", item.Name, err)
			}

			retItems = append(retItems,
				ingressWithOrder{
					ing:   item,
					order: prior,
				},
			)
			continue
		}

		// ingress no more managed -> delete finalizer
		if hasfinalizer && !managed && !HasBalancerTag(&item) {
			deletedItems = append(deletedItems, item)
			continue
		}

		// ingress is deleted or no more managed (but was) and belongs to this group -> delete finalizer
		if hasfinalizer && (!managed || deleted) && HasBalancerTag(&item) && belongsToGroup(item, tag) {
			deletedItems = append(deletedItems, item)
			continue
		}
	}

	sortIngressesByOrder(retItems)
	ret := &IngressGroup{Tag: tag, Deleted: deletedItems}
	for _, item := range retItems {
		ret.Items = append(ret.Items, item.ing)
	}

	return ret, nil
}

func belongsToGroup(ing v1.Ingress, tag string) bool {
	return GetBalancerTag(&ing) == tag
}

func parseOrder(order string) (int, error) {
	if order == "" {
		return 0, nil
	}

	res, err := strconv.Atoi(order)
	if err != nil {
		return 0, err
	}

	if res > maxAbsOrder || res < -maxAbsOrder {
		return 0, fmt.Errorf("order must be between %d and %d", -maxAbsOrder, maxAbsOrder)
	}

	return res, nil
}

type ingressWithOrder struct {
	ing   v1.Ingress
	order int
}

func sortIngressesByOrder(ings []ingressWithOrder) {
	sort.SliceStable(
		ings, func(i, j int) bool {
			if ings[i].order != ings[j].order {
				return ings[i].order < ings[j].order
			}

			if ings[i].ing.Namespace != ings[j].ing.Namespace {
				return ings[i].ing.Namespace < ings[j].ing.Namespace
			}

			return ings[i].ing.Name < ings[j].ing.Name
		},
	)
}

func IsIngressManagedByThisController(ing v1.Ingress, classes v1.IngressClassList) bool {
	if !HasBalancerTag(&ing) {
		return false
	}

	if len(classes.Items) == 0 {
		return ing.Spec.IngressClassName == nil
	}

	if ing.Spec.IngressClassName == nil {
		for _, class := range classes.Items {
			if class.Annotations[DefaultIngressClass] == "true" {
				return class.Spec.Controller == ControllerName
			}
		}
		return true
	}

	for _, class := range classes.Items {
		if *ing.Spec.IngressClassName == class.Name {
			return class.Spec.Controller == ControllerName
		}
	}
	return false
}
